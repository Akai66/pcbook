package main

import (
	"context"
	"flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
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

	//随机创建10个laptop
	for i := 0; i < 10; i++ {
		createLaptop(laptopClient)
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

func createLaptop(laptopClient pb.LaptopServiceClient) {
	laptop := sample.NewLaptop()
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
