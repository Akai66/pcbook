package service

import (
	"bytes"
	"context"
	"errors"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"pcbook/pb"
)

const (
	_maxImageSize = 1 << 20 //1M
)

type LaptopServer struct {
	LaptopStore LaptopStore
	ImageStore  ImageStore
	RateStore   RateStore
}

func NewLaptopServer(laptopStore LaptopStore, imageStore ImageStore, rateStore RateStore) *LaptopServer {
	return &LaptopServer{
		LaptopStore: laptopStore,
		ImageStore:  imageStore,
		RateStore:   rateStore,
	}
}

// CreateLaptop 创建laptop
func (server *LaptopServer) CreateLaptop(ctx context.Context, req *pb.CreateLaptopRequest) (*pb.CreateLaptopResponse, error) {
	laptop := req.GetLaptop()
	log.Printf("receive a create-laptop request with id: %s", laptop.Id)

	//生成并设置laptop.Id
	if len(laptop.Id) > 0 {
		//如果请求的laptop本身就设置了id字段，需要判断id是否为有效的uuid
		_, err := uuid.Parse(laptop.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "laptop ID is not a valid UUID: %v", err)
		}
	} else {
		//如果请求的laptop没有设置id字段，就为其随机生成一个uuid
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot generate a new laptop ID: %v", err)
		}
		laptop.Id = id.String()
	}

	//some heavy processing
	//time.Sleep(6 * time.Second)

	//判断context错误，及时停止执行，否则服务端依然会继续执行保存操作
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	//将laptop保存到内存字典中,此处使用map代替数据库
	err := server.LaptopStore.Save(laptop)
	if err != nil {
		code := codes.Internal
		if errors.Is(err, ErrAlreadyExists) {
			code = codes.AlreadyExists
		}
		return nil, status.Errorf(code, "cannot save laptop to the store: %v", err)
	}

	log.Printf("saved laptop with id: %s", laptop.Id)

	//构造响应
	res := &pb.CreateLaptopResponse{
		Id: laptop.Id,
	}
	return res, nil

}

// SearchLaptop 搜索laptop
func (server *LaptopServer) SearchLaptop(req *pb.SearchLaptopRequest, stream pb.LaptopService_SearchLaptopServer) error {
	filter := req.GetFilter()
	log.Printf("receive a search-laptop request with filter: %v", filter)

	ctx := stream.Context()
	err := server.LaptopStore.Search(ctx, filter, func(laptop *pb.Laptop) error {
		res := &pb.SearchLaptopResponse{
			Laptop: laptop,
		}
		err := stream.Send(res)
		if err != nil {
			return err
		}
		log.Printf("send laptop with id: %s", laptop.GetId())
		return nil
	})

	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error: %v", err)
	}

	return nil
}

// UploadImage 上传laptop的图片
func (server *LaptopServer) UploadImage(stream pb.LaptopService_UploadImageServer) error {
	//第一次接收图片信息
	req, err := stream.Recv()
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot receive image info: %v", err))
	}

	laptopID := req.GetInfo().GetLaptopId()
	imageType := req.GetInfo().GetImageType()
	log.Printf("receive an upload-image request for laptop %s with image type %s", laptopID, imageType)

	//查找laptop是否存在
	laptop, err := server.LaptopStore.Find(laptopID)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot find laptop: %v", err))
	}
	if laptop == nil {
		return logError(status.Errorf(codes.InvalidArgument, "laptop %s doesn't exist", laptopID))
	}

	//之后接收图片字节数据
	imageData := bytes.Buffer{}
	imageSize := 0 //图片大小，字节

	for {
		//判断context错误，在超时或客户端主动取消时，服务及时停止循环
		if err := contextErr(stream.Context()); err != nil {
			return err
		}

		log.Print("waiting to receive more data")
		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive chunk data: %v", err))
		}
		chunk := req.GetChunkData()
		size := len(chunk)
		log.Printf("receive a chunk with size: %d", size)

		imageSize += size
		//图片大小不能超过1M
		if imageSize > _maxImageSize {
			return logError(status.Errorf(codes.InvalidArgument, "image is too large:%d > %d", imageSize, _maxImageSize))
		}

		//write slow
		//time.Sleep(time.Second)

		//写入buffer
		_, err = imageData.Write(chunk)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot write chunk data: %v", err))
		}
	}

	imageID, err := server.ImageStore.Save(laptopID, imageType, imageData)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot save image to store: %v", err))
	}

	//最后服务端一次性将结果返回并关闭流
	res := &pb.UploadImageResponse{
		Id:   imageID,
		Size: uint32(imageSize),
	}
	err = stream.SendAndClose(res)
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot send response: %v", err))
	}

	log.Printf("save image with id: %s, size: %d", imageID, imageSize)

	return nil
}

// RateLaptop 提交laptop评分
func (server *LaptopServer) RateLaptop(stream pb.LaptopService_RateLaptopServer) error {
	for {
		//处理context错误
		err := contextErr(stream.Context())
		if err != nil {
			return err
		}

		//接收数据
		req, err := stream.Recv()

		if err == io.EOF {
			log.Print("no more data")
			break
		}

		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive rate laptop request: %v", err))
		}

		laptopID := req.GetLaptopId()
		score := req.GetScore()

		log.Print("receive a rate-laptop request:", req)

		//查询laptopID是否存在
		laptop, err := server.LaptopStore.Find(laptopID)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot find laptop: %v", err))
		}
		if laptop == nil {
			return logError(status.Errorf(codes.InvalidArgument, "laptop is not exist"))
		}

		//写入rate store
		rating, err := server.RateStore.Add(laptopID, score)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot add rate to store: %v", err))
		}

		//构造响应
		res := &pb.RateLaptopResponse{
			LaptopId:     laptopID,
			RatedCount:   rating.Count,
			AverageScore: rating.Sum / float64(rating.Count),
		}

		err = stream.Send(res)
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot send response: %v", err))
		}
		log.Print("send a rate-laptop response:", res)
	}

	return nil
}

// logError 记录错误日志
func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}

// contextErr 处理context错误
func contextErr(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		//客户端手动取消ctrl+c，服务端停止执行
		return logError(status.Errorf(codes.Canceled, "request is canceled"))
	case context.DeadlineExceeded:
		//超时，服务端停止执行
		return logError(status.Errorf(codes.DeadlineExceeded, "deadline is exceeded"))
	default:
		return nil
	}
}
