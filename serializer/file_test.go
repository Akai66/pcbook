package serializer

import (
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"pcbook/pb"
	"pcbook/sample"
	"testing"
)

func TestFileSerializer(t *testing.T) {
	//并发执行
	t.Parallel()
	//将proto对象序列化为二进制格式，并写入文件
	binaryFile := "../tmp/laptop.bin"
	laptop1 := sample.NewLaptop()
	err := WriteProtobufToBinaryFile(laptop1, binaryFile)
	require.NoError(t, err)

	//从文件中读取二进制数据，并反序列化为proto对象
	laptop2 := &pb.Laptop{}
	err = ReadProtobufFromBinaryFile(binaryFile, laptop2)
	require.NoError(t, err)
	require.True(t, proto.Equal(laptop1, laptop2))

	//将proto对象序列化为json格式，并写入文件
	jsonFile := "../tmp/laptop.json"
	err = WriteProtobufToJsonFile(laptop1, jsonFile)
	require.NoError(t, err)

	//从文件中读取json数据，并反序列化为proto对象
	laptop3 := &pb.Laptop{}
	err = ReadProtobufFromJsonFile(jsonFile, laptop3)
	require.NoError(t, err)
	require.True(t, proto.Equal(laptop1, laptop3))
}
