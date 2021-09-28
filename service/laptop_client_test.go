package service

import (
	"bufio"
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"io"
	"net"
	"os"
	"path/filepath"
	"pcbook/pb"
	"pcbook/sample"
	"pcbook/serializer"
	"testing"
)

func TestLaptopClient_CreateLaptop(t *testing.T) {
	t.Parallel()

	laptopStore := NewInMemoryLaptopStore()
	serverAddr := startTestLaptopServer(t, laptopStore, nil)
	laptopClient := newTestLaptopClient(t, serverAddr)

	laptop := sample.NewLaptop()
	expectedID := laptop.Id
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	other, err := laptopStore.Find(expectedID)
	require.NoError(t, err)
	require.Nil(t, other)

	res, err := laptopClient.CreateLaptop(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, expectedID, res.Id)

	//验证是否确实已经保存到存储器
	other, err = laptopStore.Find(res.Id)
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
	laptopStore := NewInMemoryLaptopStore()
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
		err := laptopStore.Save(laptop)
		require.NoError(t, err)
	}

	serverAddr := startTestLaptopServer(t, laptopStore, nil)
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

func TestLaptopClient_UploadImage(t *testing.T) {
	t.Parallel()

	testImageFolder := "../tmp" //测试图片存储文件夹

	//新建存储器
	laptopStore := NewInMemoryLaptopStore()
	imageStore := NewDiskImageStore(testImageFolder)

	//存储一个laptop
	laptop := sample.NewLaptop()
	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	//开启服务器和客户端
	serverAddr := startTestLaptopServer(t, laptopStore, imageStore)
	laptopClient := newTestLaptopClient(t, serverAddr)

	imagePath := fmt.Sprintf("%s/laptop.png", testImageFolder)
	file, err := os.Open(imagePath)
	require.NoError(t, err)
	defer file.Close()

	stream, err := laptopClient.UploadImage(context.Background())
	require.NoError(t, err)

	imageExt := filepath.Ext(imagePath)
	//第一次发送图片信息
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptop.GetId(),
				ImageType: imageExt,
			},
		},
	}
	err = stream.Send(req)
	require.NoError(t, err)

	//之后发送图片chunk data
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	size := 0
	for {
		//n表示本次一共读取了n个字节到buffer中
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				//最后一次不一定会读满1024个字节，此处必须取[:n]
				ChunkData: buffer[:n],
			},
		}
		err = stream.Send(req)
		require.NoError(t, err)
		size += n
	}

	//最后结束流，并接收服务端响应数据
	res, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.NotEmpty(t, res.GetId())
	//qual必须二者类型和值都相等，才符合;EqualValues，值相等即符合;
	require.EqualValues(t, size, res.GetSize())

	targetImagePath := fmt.Sprintf("%s/%s%s", testImageFolder, res.GetId(), imageExt)
	require.FileExists(t, targetImagePath)

	//删除测试图片
	err = os.Remove(targetImagePath)
	require.NoError(t, err)
}

// startTestLaptopServer 启动一个测试的grpc服务器
func startTestLaptopServer(t *testing.T, laptopStore LaptopStore, imageStore ImageStore) string {
	laptopServer := NewLaptopServer(laptopStore, imageStore)
	grpcServer := grpc.NewServer()
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)
	listener, err := net.Listen("tcp", ":0") //随机监听一个可用的端口
	require.NoError(t, err)

	//独立协程启动server
	go grpcServer.Serve(listener)
	return listener.Addr().String()
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
