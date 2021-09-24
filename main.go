package main

import (
	"fmt"
	"pcbook/pb"
	"pcbook/serializer"
)

func main() {
	laptop := &pb.Laptop{}
	err := serializer.ReadProtobufFromJsonFile("./tmp/laptop.json", laptop)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%v", laptop.Weight.(*pb.Laptop_WeightKg).WeightKg)
}
