package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"pcbook/pb"
)

type LaptopServer struct {
	Store LaptopStore
}

func NewLaptopServer(store LaptopStore) *LaptopServer {
	return &LaptopServer{Store: store}
}

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
	if ctx.Err() == context.Canceled {
		//客户端手动取消ctrl+c，服务端停止执行
		return nil, status.Errorf(codes.Canceled, "request is canceled")
	}

	if ctx.Err() == context.DeadlineExceeded {
		//超时，服务端停止执行
		return nil, status.Errorf(codes.DeadlineExceeded, "deadline is exceeded")
	}

	//将laptop保存到内存字典中,此处使用map代替数据库
	err := server.Store.Save(laptop)
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

func (server *LaptopServer) SearchLaptop(req *pb.SearchLaptopRequest, stream pb.LaptopService_SearchLaptopServer) error {
	filter := req.GetFilter()
	log.Printf("receive a search-laptop request with filter: %v", filter)

	ctx := stream.Context()
	err := server.Store.Search(ctx, filter, func(laptop *pb.Laptop) error {
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
