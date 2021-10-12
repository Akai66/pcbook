package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"pcbook/pb"
	"pcbook/service"
	"time"
)

const (
	_secretKey         = "secret"         //用于签名和验证token的密钥
	_tokenDuration     = 15 * time.Minute //token有效时长
	_laptopServicePath = "/LaptopService/"
	_serverPem         = "cert/server.pem"
	_serverKey         = "cert/server.key"
	_caPem             = "cert/ca.pem"
)

// accessibleRoles 方法路径及对应的有访问权限的角色列表
func accessibleRoles() map[string][]string {
	return map[string][]string{
		_laptopServicePath + "CreateLaptop": {"admin"},
		_laptopServicePath + "UploadImage":  {"admin"},
		_laptopServicePath + "RateLaptop":   {"admin", "user"},
	}
}

// seedUsers 模拟生成管理员及普通用户
func seedUsers(userStore service.UserStore) error {
	err := createUser(userStore, "admin1", "secret", "admin")
	if err != nil {
		return err
	}
	return createUser(userStore, "user1", "secret", "user")
}

// createUser 新建user并保存至userStore
func createUser(userStore service.UserStore, username, password, role string) error {
	user, err := service.NewUser(username, password, role)
	if err != nil {
		return err
	}
	return userStore.Save(user)
}

// loadTLSCredentials 加载服务端证书，私钥，以及ca根证书
func loadTLSCredentials() (credentials.TransportCredentials, error) {
	//加载服务端证书
	cert, err := tls.LoadX509KeyPair(_serverPem, _serverKey)
	if err != nil {
		return nil, err
	}
	//加载ca证书，双向验证，服务端ca证书主要用来验证客户端证书是否合法
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(_caPem)
	if err != nil {
		return nil, err
	}
	certPool.AppendCertsFromPEM(ca)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert}, //服务端证书
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})
	return creds, nil
}

// runGRPCServer start grpc server
func runGRPCServer(authServer pb.AuthServiceServer, laptopServer pb.LaptopServiceServer, jwtManager *service.JWTManager, enableTLS bool, listener net.Listener) error {
	//生成拦截器
	interceptor := service.NewAuthInterceptor(jwtManager, accessibleRoles())

	serverOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.Unary()),   //添加普通模式拦截器
		grpc.StreamInterceptor(interceptor.Stream()), //添加流模式拦截器
	}

	if enableTLS {
		//加载TLS证书
		creds, err := loadTLSCredentials()
		if err != nil {
			return fmt.Errorf("cannot load TLSCredentials: %v", err)
		}
		serverOptions = append(serverOptions, grpc.Creds(creds))
	}
	grpcServer := grpc.NewServer(serverOptions...)
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	pb.RegisterAuthServiceServer(grpcServer, authServer)

	log.Printf("start GRPC server on port: %s, TLS = %t", listener.Addr().String(), enableTLS)
	return grpcServer.Serve(listener)
}

// runRESTServer start rest server
func runRESTServer(authServer pb.AuthServiceServer, laptopServer pb.LaptopServiceServer, enableTLS bool, listener net.Listener, grpcEndpoint string) error {
	mux := runtime.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//若grpc server启用了tls验证，也需要在此处添加客户端tls证书，grpc.WithTransportCredentials(creds)
	dialOption := []grpc.DialOption{grpc.WithInsecure()}
	//err := pb.RegisterAuthServiceHandlerServer(ctx, mux, authServer) //仅支持普通模式，只需要启动http server
	err := pb.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, dialOption) //支持普通模式和流模式，需要同时启动http server和grpc server
	if err != nil {
		return err
	}

	//err = pb.RegisterLaptopServiceHandlerServer(ctx, mux, laptopServer) //仅支持普通模式，只需要启动http server
	err = pb.RegisterLaptopServiceHandlerFromEndpoint(ctx, mux, grpcEndpoint, dialOption) //支持普通模式和流模式，需要同时启动http server和grpc server
	if err != nil {
		return err
	}

	log.Printf("start REST server on port: %s, TLS = %t", listener.Addr().String(), enableTLS)
	if enableTLS {
		return http.ServeTLS(listener, mux, _serverPem, _serverKey)
	}
	return http.Serve(listener, mux)
}

func main() {
	port := flag.Int("port", 0, "rpc server port")
	enableTLS := flag.Bool("tls", false, "enable SSL/TLS")
	serverType := flag.String("type", "grpc", "type of server (grpc/rest)")
	endPoint := flag.String("endpoint", "", "grpc endpoint")
	flag.Parse()

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

	address := fmt.Sprintf("0.0.0.0:%d", *port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}

	if *serverType == "grpc" {
		err = runGRPCServer(authServer, laptopServer, jwtManager, *enableTLS, listener)
	} else {
		err = runRESTServer(authServer, laptopServer, *enableTLS, listener, *endPoint)
	}

	if err != nil {
		log.Fatalf("cannot start server: %v", err)
	}

}
