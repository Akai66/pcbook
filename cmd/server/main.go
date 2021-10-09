package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"pcbook/pb"
	"pcbook/service"
	"time"
)

const (
	_secretKey         = "secret"         //用于签名和验证token的密钥
	_tokenDuration     = 15 * time.Minute //token有效时长
	_laptopServicePath = "/LaptopService/"
)

// accessibleRoles 方法路径及对应的有访问权限的角色列表
func accessibleRoles() map[string][]string {
	return map[string][]string{
		_laptopServicePath + "CreateLaptop": {"admin"},
		_laptopServicePath + "UploadImage":  {"admin"},
		_laptopServicePath + "RateLaptop":   {"admin", "user"},
	}
}

// seedUsers 生成管理员及普通用户
func seedUsers(userStore service.UserStore) error {
	err := createUser(userStore, "admin1", "secret", "admin")
	if err != nil {
		return err
	}
	return createUser(userStore, "user1", "secret", "user")
}

func createUser(userStore service.UserStore, username, password, role string) error {
	user, err := service.NewUser(username, password, role)
	if err != nil {
		return err
	}
	return userStore.Save(user)
}

func main() {
	port := flag.Int("port", 0, "rpc server port")
	flag.Parse()
	log.Printf("start server on port: %d", *port)

	userStore := service.NewInMemoryUserStore()
	err := seedUsers(userStore)
	if err != nil {
		log.Fatal("cannot seed users")
	}
	jwtManager := service.NewJWTManager(_secretKey, _tokenDuration)
	authServer := service.NewAuthServer(userStore, jwtManager)

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("img")
	rateStore := service.NewInMemoryRateStore()
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, rateStore)

	//生成拦截器
	interceptor := service.NewAuthInterceptor(jwtManager, accessibleRoles())

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.Unary()),   //添加普通模式拦截器
		grpc.StreamInterceptor(interceptor.Stream()), //添加流模式拦截器
	)
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	pb.RegisterAuthServiceServer(grpcServer, authServer)

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
