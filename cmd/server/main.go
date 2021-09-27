package main

import (
	"google.golang.org/grpc"
	"net"
	"pcbook/pb"
	"pcbook/service"
)

func main() {
	laptopServer := service.NewLaptopServer(service.NewInMemoryLaptopStore())
	grpcServer := grpc.NewServer()
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)
	listener, _ := net.Listen("tcp", ":8080") //随机监听一个可用的端口
	grpcServer.Serve(listener)
}
