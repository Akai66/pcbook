package main

import (
	"bufio"
	"context"
	"flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
	"path/filepath"
	"pcbook/pb"
	"pcbook/sample"
	"time"
)

func main() {
	serverAddr := flag.String("address", "", "rpc server address")
	flag.Parse()
	log.Printf("dial server %s", *serverAddr)

	conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("cannot dial server: %v", err)
	}
	laptopClient := pb.NewLaptopServiceClient(conn)
	testUploadImage(laptopClient)
}

func testCreateLaptop(laptopClient pb.LaptopServiceClient) {
	createLaptop(laptopClient, sample.NewLaptop())
}

func testSearchLaptop(laptopClient pb.LaptopServiceClient) {
	//随机创建10个laptop
	for i := 0; i < 10; i++ {
		createLaptop(laptopClient, sample.NewLaptop())
	}

	//构造筛选条件
	filter := &pb.Filter{
		MaxPriceUsed: 3000,
		MinCpuCores:  4,
		MinCpuGhz:    2.5,
		MinRam:       &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}
	//从服务端查询laptop
	searchLaptop(laptopClient, filter)
}

func testUploadImage(laptopClient pb.LaptopServiceClient) {
	laptop := sample.NewLaptop()
	createLaptop(laptopClient, laptop)
	uploadImage(laptopClient, laptop.Id, "tmp/laptop.png")
}

func createLaptop(laptopClient pb.LaptopServiceClient, laptop *pb.Laptop) {
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	//如果server响应时间超过5秒则返回DeadlineExceeded err
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := laptopClient.CreateLaptop(ctx, req)
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

func searchLaptop(laptopClient pb.LaptopServiceClient, filter *pb.Filter) {
	log.Printf("search filter: %v", filter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.SearchLaptopRequest{
		Filter: filter,
	}

	stream, err := laptopClient.SearchLaptop(ctx, req)
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

func uploadImage(laptopClient pb.LaptopServiceClient, laptopID, imagePath string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal("cannot open image file:", err)
	}
	defer file.Close()

	//如果server响应时间超过5秒则返回DeadlineExceeded err
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.UploadImage(ctx)
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
