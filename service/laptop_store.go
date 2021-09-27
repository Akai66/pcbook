package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/jinzhu/copier"
	"log"
	"pcbook/pb"
	"sync"
	"time"
)

var ErrAlreadyExists = errors.New("record already exists")

//LaptopStore 笔记本存储器接口
type LaptopStore interface {
	//Save 保存数据
	Save(laptop *pb.Laptop) error
	//Find 根据id查找laptop
	Find(id string) (*pb.Laptop, error)
	//Search 根据filter筛选，其中found为回调函数
	Search(ctx context.Context, filter *pb.Filter, found func(laptop *pb.Laptop) error) error
}

//InMemoryLaptopStore 内存存储器，使用map存储
type InMemoryLaptopStore struct {
	mutex sync.RWMutex //开箱即用的
	data  map[string]*pb.Laptop
}

func NewInMemoryLaptopStore() *InMemoryLaptopStore {
	return &InMemoryLaptopStore{
		data: make(map[string]*pb.Laptop),
	}
}

func (store *InMemoryLaptopStore) Save(laptop *pb.Laptop) error {
	//加写锁
	store.mutex.Lock()
	defer store.mutex.Unlock()
	//id已存在，则返回错误
	if store.data[laptop.Id] != nil {
		return ErrAlreadyExists
	}
	//由于形参和map的value都是指针类型，为了防止外部指针变量指向的内容被修改时，影响到已存储在map中的value
	//需要利用copier进行deep copy，进而把二者彻底分开

	other, err := deepCopy(laptop)
	if err != nil {
		return err
	}
	store.data[other.Id] = other
	return nil
}

func (store *InMemoryLaptopStore) Find(id string) (*pb.Laptop, error) {
	//加读锁
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	laptop := store.data[id]
	if laptop == nil {
		return nil, nil
	}

	//此处同上，也是为了防止map中value的内容被外部修改，需要进行深拷贝
	return deepCopy(laptop)
}

func (store *InMemoryLaptopStore) Search(ctx context.Context, filter *pb.Filter, found func(laptop *pb.Laptop) error) error {
	//加读锁
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	//遍历筛选符合条件的laptop
	for _, laptop := range store.data {
		//heavy processing
		time.Sleep(1 * time.Second)
		log.Printf("checking laptop id: %s", laptop.GetId())
		//如果超时或客户端ctrl+c,则结束循环,避免浪费服务器资源
		if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
			log.Print("context is canceled")
			return errors.New("context is canceled")
		}
		if isQualified(filter, laptop) {
			//deep copy
			other, err := deepCopy(laptop)
			if err != nil {
				return err
			}
			//调用回调函数
			err = found(other)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//isQualified 判断laptop是否满足filter的要求
func isQualified(filter *pb.Filter, laptop *pb.Laptop) bool {
	if laptop.GetPriceUsed() > filter.GetMaxPriceUsed() {
		return false
	}
	if laptop.GetCpu().GetNumberCores() < filter.GetMinCpuCores() {
		return false
	}
	if laptop.GetCpu().GetMinGhz() < filter.GetMinCpuGhz() {
		return false
	}
	if toBit(laptop.GetRam()) < toBit(filter.GetMinRam()) {
		return false
	}
	return true
}

//toBit 将内存大小单位转换为bit
func toBit(memory *pb.Memory) uint64 {
	value := memory.GetValue()
	switch memory.GetUnit() {
	case pb.Memory_BIT:
		return value
	case pb.Memory_BYTE:
		return value << 3
	case pb.Memory_KILOBYTE:
		return value << 13
	case pb.Memory_MEGABYTE:
		return value << 23
	case pb.Memory_GIGABYTE:
		return value << 33
	case pb.Memory_TERABYTE:
		return value << 43
	default:
		return 0
	}
}

func deepCopy(laptop *pb.Laptop) (*pb.Laptop, error) {
	other := &pb.Laptop{}
	err := copier.Copy(other, laptop)
	if err != nil {
		return nil, fmt.Errorf("cannot copy laptop data:%w", err)
	}
	return other, nil
}
