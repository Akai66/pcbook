package client

import (
	"bufio"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
	"path/filepath"
	"pcbook/pb"
	"time"
)

type LaptopClient struct {
	service pb.LaptopServiceClient
}

func NewLaptopClient(cc *grpc.ClientConn) *LaptopClient {
	service := pb.NewLaptopServiceClient(cc)
	return &LaptopClient{service}
}

// CreateLaptop 新建laptop
func (client *LaptopClient) CreateLaptop(laptop *pb.Laptop) {
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	//如果server响应时间超过5秒则返回DeadlineExceeded err
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.service.CreateLaptop(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Print("laptop already exists")
		} else {
			log.Fatalf("cannot create laptop: %v", err)
		}
		return
	}

	log.Printf("created laptop with id: %s", res.Id)
}

// SearchLaptop 服务端流模式，根据条件筛选符合要求的laptop
func (client *LaptopClient) SearchLaptop(filter *pb.Filter) {
	log.Printf("search filter: %v", filter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.SearchLaptopRequest{
		Filter: filter,
	}

	stream, err := client.service.SearchLaptop(ctx, req)
	if err != nil {
		log.Fatalf("cannot search laptop: %v", err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("cannot receive response: %v", err)
		}
		laptop := res.GetLaptop()
		log.Print("- found: ", laptop.GetId())
		log.Print("	 + brand: ", laptop.GetBrand())
		log.Print("	 + name: ", laptop.GetName())
		log.Print("	 + price: ", laptop.GetPriceUsed())
		log.Print("	 + cpu cores: ", laptop.GetCpu().GetNumberCores())
		log.Print("	 + cpu min ghz: ", laptop.GetCpu().GetMinGhz())
		log.Print("	 + ram: ", laptop.GetRam().GetValue(), laptop.GetRam().GetUnit())
	}
}

// UploadImage 客户端流模式，分chunk上传图片
func (client *LaptopClient) UploadImage(laptopID, imagePath string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal("cannot open image file:", err)
	}
	defer file.Close()

	//如果server响应时间超过5秒则返回DeadlineExceeded err
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.service.UploadImage(ctx)
	if err != nil {
		log.Fatal("cannot upload image:", err)
	}

	//第一次发送图片信息
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptopID,
				ImageType: filepath.Ext(imagePath),
			},
		},
	}
	err = stream.Send(req)
	if err != nil {
		//当服务端发生错误时，会直接关闭流，那么在客户端得到的错误就是EOF，无法获取具体的错误信息
		//需要使用stream.RecvMsg(nil)获取服务端具体的grpc错误信息
		log.Fatal("cannot send image info:", err, stream.RecvMsg(nil))
	}

	//之后发送图片chunk data
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	for {
		//n表示本次一共读取了n个字节到buffer中
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer:", err)
		}
		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				//最后一次不一定会读满1024个字节，此处必须取[:n]
				ChunkData: buffer[:n],
			},
		}
		err = stream.Send(req)
		if err != nil {
			//当服务端发生错误时，会直接关闭流，那么在客户端得到的错误就是EOF，无法获取具体的错误信息
			//需要使用stream.RecvMsg(nil)获取服务端具体的grpc错误信息
			log.Fatal("cannot send image chunk data:", err, stream.RecvMsg(nil))
		}
	}

	//最后结束流，并接收服务端响应数据
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("cannot receive response:", err)
	}

	log.Printf("image uploaded with id: %s, size: %d", res.GetId(), res.GetSize())
}

// RateLaptop 双向流模式，提交laptop的评分
func (client *LaptopClient) RateLaptop(laptopIDs []string, scores []float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.service.RateLaptop(ctx)
	if err != nil {
		log.Fatal("cannot rate laptop:", err)
	}

	//单独起一个go程接收并处理响应
	waitResponse := make(chan error)
	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				waitResponse <- nil
				return
			}
			if err != nil {
				waitResponse <- err
				return
			}
			log.Print("receive res:", res)
		}
	}()

	//发送请求
	for i, laptopID := range laptopIDs {
		req := &pb.RateLaptopRequest{
			LaptopId: laptopID,
			Score:    scores[i],
		}
		err := stream.Send(req)
		if err != nil {
			log.Fatal("cannot send rate request:", err, stream.RecvMsg(nil))
		}
		log.Print("send req:", req)
	}

	err = stream.CloseSend()
	if err != nil {
		log.Fatal("close stream failed:", err)
	}

	err = <-waitResponse
	if err != nil {
		log.Fatal("receive response failed:", err)
	}
}
