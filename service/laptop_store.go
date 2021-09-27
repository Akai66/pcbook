package service

import (
	"errors"
	"fmt"
	"github.com/jinzhu/copier"
	"pcbook/pb"
	"sync"
)

var ErrAlreadyExists = errors.New("record already exists")

//LaptopStore 笔记本存储器接口
type LaptopStore interface {
	//Save 保存数据
	Save(laptop *pb.Laptop) error
	//Find 根据id查找laptop
	Find(id string) (*pb.Laptop, error)
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
	other := &pb.Laptop{}
	err := copier.Copy(other, laptop)
	if err != nil {
		return fmt.Errorf("cannot copy laptop data:%w", err)
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
	other := &pb.Laptop{}
	err := copier.Copy(other, laptop)
	if err != nil {
		return nil, fmt.Errorf("cannot copy laptop data:%w", err)
	}
	return other, nil
}
