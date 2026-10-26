package repository

import (
	"sync"

	"api/internal/model"
)

type Store interface {
	View(fn func(users map[string]*model.Wallet) error) error
	Update(fn func(users map[string]*model.Wallet) error) error
}

type MemoryStore struct {
	mu    sync.RWMutex
	users map[string]*model.Wallet
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users: make(map[string]*model.Wallet),
	}
}

func (s *MemoryStore) View(fn func(users map[string]*model.Wallet) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fn(s.users)
}

func (s *MemoryStore) Update(fn func(users map[string]*model.Wallet) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(s.users)
}
