package serializer

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

func ProtobufToJson(message proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		EnumsAsInts:  false, //枚举类型是否用整型表示
		EmitDefaults: true,  //是否写入具有默认值的字段
		Indent:       "  ",  //json的缩进符
		OrigName:     true,  //是否使用proto原型文件中定义的原始字段名
	}
	return marshaler.MarshalToString(message)
}

func JsonToProtobuf(jsonStr string, message proto.Message) error {
	return jsonpb.UnmarshalString(jsonStr, message)
}
