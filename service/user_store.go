package service

import (
	"sync"
)

type UserStore interface {
	Save(user *User) error
	Find(username string) (*User, error)
}

type InMemoryUserStore struct {
	mutex sync.RWMutex
	users map[string]*User
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users: make(map[string]*User),
	}
}

// Save 保存User对象
func (store *InMemoryUserStore) Save(user *User) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.users[user.Username] != nil {
		return ErrAlreadyExists
	}
	other := user.Clone()
	store.users[user.Username] = other
	return nil
}

// Find 查找User对象
func (store *InMemoryUserStore) Find(username string) (*User, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	user := store.users[username]
	if user == nil {
		return nil, nil
	}
	other := user.Clone()
	return other, nil
}
