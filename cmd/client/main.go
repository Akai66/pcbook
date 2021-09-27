package main

import (
	"context"
	"flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
