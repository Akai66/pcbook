package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"pcbook/pb"
	"pcbook/sample"
)

func main() {
	conn, _ := grpc.Dial(":8080", grpc.WithInsecure())
	laptopClient := pb.NewLaptopServiceClient(conn)
	req := &pb.CreateLaptopRequest{
		Laptop: sample.NewLaptop(),
	}
	res, _ := laptopClient.CreateLaptop(context.Background(), req)
	fmt.Println(res)
}
