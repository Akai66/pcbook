package service

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"io"
	"net"
	"pcbook/pb"
	"pcbook/sample"
	"pcbook/serializer"
	"testing"
)

func TestLaptopClient_CreateLaptop(t *testing.T) {
	t.Parallel()

	laptopServer, serverAddr := startTestLaptopServer(t, NewInMemoryLaptopStore())
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

func TestLaptopClient_SearchLaptop(t *testing.T) {
	t.Parallel()

	filter := &pb.Filter{
		MaxPriceUsed: 3000,
		MinCpuCores:  4,
		MinCpuGhz:    2.5,
		MinRam:       &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}
	store := NewInMemoryLaptopStore()
	expectedIDs := make(map[string]bool)

	for i := 0; i < 6; i++ {
		laptop := sample.NewLaptop()
		switch i {
		case 0:
			laptop.PriceUsed = 3100
		case 1:
			laptop.Cpu.NumberCores = 2
		case 2:
			laptop.Cpu.MinGhz = 2.4
		case 3:
			laptop.Ram = &pb.Memory{Value: 4096, Unit: pb.Memory_MEGABYTE}
		case 4:
			laptop.PriceUsed = 2999.99
			laptop.Cpu.NumberCores = 6
			laptop.Cpu.MinGhz = 2.6
			laptop.Ram = &pb.Memory{Value: 9, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		case 5:
			laptop.PriceUsed = 2800
			laptop.Cpu.NumberCores = 8
			laptop.Cpu.MinGhz = 3.1
			laptop.Ram = &pb.Memory{Value: 10, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		}
		err := store.Save(laptop)
		require.NoError(t, err)
	}

	_, serverAddr := startTestLaptopServer(t, store)
	laptopClient := newTestLaptopClient(t, serverAddr)
	req := &pb.SearchLaptopRequest{
		Filter: filter,
	}
	stream, err := laptopClient.SearchLaptop(context.Background(), req)
	require.NoError(t, err)
	count := 0

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Contains(t, expectedIDs, res.GetLaptop().GetId())

		count++
	}

	require.Equal(t, len(expectedIDs), count)
}

// startTestLaptopServer 启动一个测试的grpc服务器
func startTestLaptopServer(t *testing.T, store LaptopStore) (*LaptopServer, string) {
	laptopServer := NewLaptopServer(store)
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
	//require.Equal(t, laptop1, laptop2)
	//通过proto.Equal()方法比较，二者是相等的
	require.True(t, proto.Equal(laptop1, laptop2))
	//通过将proto对象转换为json字符串比较，也是相等的
	json1, err := serializer.ProtobufToJson(laptop1)
	require.NoError(t, err)
	json2, err := serializer.ProtobufToJson(laptop2)
	require.NoError(t, err)
	require.Equal(t, json1, json2)
}
