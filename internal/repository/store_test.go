package repository

import (
	"errors"
	"sync"
	"testing"

	"api/internal/model"
)

func TestUpdateThenView(t *testing.T) {
	s := NewMemoryStore()

	err := s.Update(func(users map[string]*model.Wallet) error {
		users["alice"] = &model.Wallet{Balance: 10}
		return nil
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	var got float32
	err = s.View(func(users map[string]*model.Wallet) error {
		got = users["alice"].Balance
		return nil
	})
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	if got != 10 {
		t.Errorf("balance = %v, want 10", got)
	}
}

func TestErrorsArePassedThrough(t *testing.T) {
	s := NewMemoryStore()
	boom := errors.New("boom")

	if err := s.View(func(map[string]*model.Wallet) error { return boom }); err != boom {
		t.Errorf("View err = %v, want %v", err, boom)
	}
	if err := s.Update(func(map[string]*model.Wallet) error { return boom }); err != boom {
		t.Errorf("Update err = %v, want %v", err, boom)
	}
}

func TestConcurrentUpdates(t *testing.T) {
	s := NewMemoryStore()
	s.Update(func(users map[string]*model.Wallet) error {
		users["alice"] = &model.Wallet{}
		return nil
	})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Update(func(users map[string]*model.Wallet) error {
				users["alice"].Balance++
				return nil
			})
		}()
	}
	wg.Wait()

	s.View(func(users map[string]*model.Wallet) error {
		if users["alice"].Balance != 100 {
			t.Errorf("balance = %v, want 100", users["alice"].Balance)
		}
		return nil
	})
}
