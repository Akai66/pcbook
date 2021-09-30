package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"pcbook/pb"
	"pcbook/service"
)

func main() {
	port := flag.Int("port", 0, "rpc server port")
	flag.Parse()
	log.Printf("start server on port: %d", *port)
	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("img")
	rateStore := service.NewInMemoryRateStore()
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, rateStore)
	grpcServer := grpc.NewServer()
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	address := fmt.Sprintf("0.0.0.0:%d", *port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}
}
