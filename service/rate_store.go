package service

import (
	"fmt"
	"github.com/jinzhu/copier"
	"sync"
)

type RateStore interface {
	Add(laptopID string, score float64) (*Rating, error)
}

type Rating struct {
	Count uint32
	Sum   float64
}

type InMemoryRateStore struct {
	mutex sync.RWMutex
	data  map[string]*Rating
}

func NewInMemoryRateStore() *InMemoryRateStore {
	return &InMemoryRateStore{
		data: make(map[string]*Rating),
	}
}

func (store *InMemoryRateStore) Add(laptopID string, score float64) (*Rating, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	rating := store.data[laptopID]
	if rating == nil {
		rating = &Rating{
			Count: 1,
			Sum:   score,
		}
	} else {
		rating.Count++
		rating.Sum += score
	}
	store.data[laptopID] = rating
	//deep copy
	return deepCopyRating(rating)

}

func deepCopyRating(rating *Rating) (*Rating, error) {
	other := &Rating{}
	err := copier.Copy(other, rating)
	if err != nil {
		return nil, fmt.Errorf("cannot copy rating data:%w", err)
	}
	return other, nil
}
