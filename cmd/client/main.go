package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"pcbook/client"
	"pcbook/pb"
	"pcbook/sample"
	"strings"
	"time"
)

const (
	_username          = "admin1"
	_password          = "secret"
	_refreshDuration   = 30 * time.Second
	_laptopServicePath = "/LaptopService/"
)

// authMethods 需要鉴权验证的方法
func authMethods() map[string]bool {
	return map[string]bool{
		_laptopServicePath + "CreateLaptop": true,
		_laptopServicePath + "UploadImage":  true,
		_laptopServicePath + "RateLaptop":   true,
	}
}

func main() {
	serverAddr := flag.String("address", "", "rpc server address")
	flag.Parse()
	log.Printf("dial server %s", *serverAddr)

	//先创建auth客户端连接，用于获取token
	conn1, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("cannot dial server: %v", err)
	}
	authClient := client.NewAuthClient(conn1, _username, _password)
	//testLogin(authClient)

	//通过auth客户端连接，构造客户端鉴权验证拦截器
	interceptor, err := client.NewAuthInterceptor(authClient, authMethods(), _refreshDuration)
	if err != nil {
		log.Fatalf("cannot create auth interceptor: %v", err)
	}

	//最后创建laptop客户端连接，并绑定客户端拦截器方法，客户端每次执行rpc调用时，会先调用拦截器方法，在拦截器方法中将最新的token添加到context中
	conn2, err := grpc.Dial(
		*serverAddr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	)
	laptopClient := client.NewLaptopClient(conn2)
	//testUploadImage(laptopClient)
	testRateLaptop(laptopClient)

}

func testCreateLaptop(laptopClient *client.LaptopClient) {
	laptopClient.CreateLaptop(sample.NewLaptop())
}

func testSearchLaptop(laptopClient *client.LaptopClient) {
	//随机创建10个laptop
	for i := 0; i < 10; i++ {
		laptopClient.CreateLaptop(sample.NewLaptop())
	}

	//构造筛选条件
	filter := &pb.Filter{
		MaxPriceUsed: 3000,
		MinCpuCores:  4,
		MinCpuGhz:    2.5,
		MinRam:       &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}
	//从服务端查询laptop
	laptopClient.SearchLaptop(filter)
}

func testUploadImage(laptopClient *client.LaptopClient) {
	laptop := sample.NewLaptop()
	laptopClient.CreateLaptop(laptop)
	laptopClient.UploadImage(laptop.Id, "tmp/laptop.png")
}

func testRateLaptop(laptopClient *client.LaptopClient) {
	count := 3
	laptopIDs := make([]string, count)
	scores := make([]float64, count)
	for i := 0; i < count; i++ {
		laptop := sample.NewLaptop()
		laptopClient.CreateLaptop(laptop)
		laptopIDs[i] = laptop.Id
	}

	//用户循环确认是否随机提交评分
	for {
		fmt.Print("rate laptop (y/n)?")
		var answer string
		fmt.Scan(&answer)
		if strings.ToLower(answer) != "y" {
			break
		}
		for i := 0; i < count; i++ {
			scores[i] = sample.RandomLaptopScore()
		}
		laptopClient.RateLaptop(laptopIDs, scores)
	}

}

func testLogin(authClient *client.AuthClient) {
	token, err := authClient.Login()
	if err != nil {
		log.Fatalf("cannot login: %v", err)
	}
	log.Printf("accessToken:%v", token)
}
