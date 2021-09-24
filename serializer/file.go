package serializer

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
)

// WriteProtobufToBinaryFile 将proto数据对象按照protobuf协议序列化为二进制格式后，写入文件
func WriteProtobufToBinaryFile(message proto.Message, filename string) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("cannot marshal proto message to binary:%w", err)
	}
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("cannot write binary data to file:%w", err)
	}
	return nil
}

// ReadProtobufFromBinaryFile 从文件中读取二进制数据，并反序列化为proto对象
func ReadProtobufFromBinaryFile(filename string, message proto.Message) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read binary data from file:%w", err)
	}
	err = proto.Unmarshal(data, message)
	if err != nil {
		return fmt.Errorf("cannot unmarshal proto message from binary:%w", err)
	}
	return nil
}

// WriteProtobufToJsonFile 将proto数据对象序列化为json格式，写入文件
func WriteProtobufToJsonFile(message proto.Message, filename string) error {
	data, err := ProtobufToJson(message)
	if err != nil {
		return fmt.Errorf("cannot marshal proto to json:%w", err)
	}
	err = ioutil.WriteFile(filename, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("cannot write json data to file:%w", err)
	}
	return nil
}

// ReadProtobufFromJsonFile 从文件中读取json数据，并反序列化为proto对象
func ReadProtobufFromJsonFile(filename string, message proto.Message) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read json data from file:%w", err)
	}
	err = JsonToProtobuf(string(data), message)
	if err != nil {
		return fmt.Errorf("cannot unmarshal proto message from json:%w", err)
	}
	return err
}
