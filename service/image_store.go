package service

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"os"
	"sync"
)

// ImageStore 图片存储器接口
type ImageStore interface {
	Save(laptopID string, imageType string, imageData bytes.Buffer) (string, error)
}

// DiskImageStore 硬盘存储器实现
type DiskImageStore struct {
	mutex       sync.RWMutex
	imageFolder string                //图片保存文件夹路径
	images      map[string]*ImageInfo //图片的信息
}

// ImageInfo 图片信息
type ImageInfo struct {
	LaptopID string
	Type     string
	Path     string
}

func NewDiskImageStore(imageFolder string) *DiskImageStore {
	return &DiskImageStore{
		imageFolder: imageFolder,
		images:      make(map[string]*ImageInfo),
	}
}

func (store *DiskImageStore) Save(laptopID string, imageType string, imageData bytes.Buffer) (string, error) {
	//1.将图片写入文件中

	//先使用uuid随机生成图片id
	imageID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("cannot generate image id: %w", err)
	}

	imagePath := fmt.Sprintf("%s/%s%s", store.imageFolder, imageID, imageType)

	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("cannot create image file: %w", err)
	}

	_, err = imageData.WriteTo(file)
	if err != nil {
		return "", fmt.Errorf("cannot write image to file: %w", err)
	}

	//2.将图片信息存储到map中
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	store.images[imageID.String()] = &ImageInfo{
		LaptopID: laptopID,
		Type:     imageType,
		Path:     imagePath,
	}

	return imageID.String(), nil
}
