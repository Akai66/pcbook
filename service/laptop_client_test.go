package service

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"net"
	"pcbook/pb"
	"pcbook/sample"
	"pcbook/serializer"
	"testing"
)

func TestLaptopClient_CreateLaptop(t *testing.T) {
	t.Parallel()

	laptopServer, serverAddr := startTestLaptopServer(t)
	laptopClient := newTestLaptopClient(t, serverAddr)

	laptop := sample.NewLaptop()
	expectedID := laptop.Id
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	other, err := laptopServer.Store.Find(expectedID)
	require.NoError(t, err)
	require.Nil(t, other)

	res, err := laptopClient.CreateLaptop(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, expectedID, res.Id)

	//验证是否确实已经保存到存储器
	other, err = laptopServer.Store.Find(res.Id)
	require.NoError(t, err)
	require.NotNil(t, other)

	//验证两个laptop是否相等
	requireSameLaptop(t, laptop, other)
}

// startTestLaptopServer 启动一个测试的grpc服务器
func startTestLaptopServer(t *testing.T) (*LaptopServer, string) {
	laptopServer := NewLaptopServer(NewInMemoryLaptopStore())
	grpcServer := grpc.NewServer()
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)
	listener, err := net.Listen("tcp", ":0") //随机监听一个可用的端口
	require.NoError(t, err)

	//独立协程启动server
	go grpcServer.Serve(listener)
	return laptopServer, listener.Addr().String()
}

// newTestLaptopClient 创建一个客户端
func newTestLaptopClient(t *testing.T, serverAddr string) pb.LaptopServiceClient {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	require.NoError(t, err)
	return pb.NewLaptopServiceClient(conn)
}

func requireSameLaptop(t *testing.T, laptop1, laptop2 *pb.Laptop) {
	//这里二者是不相等的，因为在结构体内部有一些特殊领域，用于序列化对象使用
	require.Equal(t, laptop1, laptop2)
	//通过proto.Equal()方法比较，二者是相等的
	require.True(t, proto.Equal(laptop1, laptop2))
	//通过将proto对象转换为json字符串比较，也是相等的
	json1, err := serializer.ProtobufToJson(laptop1)
	require.NoError(t, err)
	json2, err := serializer.ProtobufToJson(laptop2)
	require.NoError(t, err)
	require.Equal(t, json1, json2)
}
