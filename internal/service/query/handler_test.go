package query

import (
	"errors"
	"testing"

	"api/internal/model"
	"api/internal/repository"
)

func newHandler() (*Handler, *repository.MemoryStore) {
	store := repository.NewMemoryStore()
	store.Update(func(users map[string]*model.Wallet) error {
		users["alice"] = &model.Wallet{Balance: 42}
		return nil
	})
	return NewHandler(store), store
}

func TestGetBalance(t *testing.T) {
	h, _ := newHandler()

	w, err := h.GetBalance(GetBalance{Username: "alice"})
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if w.Balance != 42 {
		t.Errorf("balance = %v, want 42", w.Balance)
	}
}

func TestGetBalanceUnknownUser(t *testing.T) {
	h, _ := newHandler()

	_, err := h.GetBalance(GetBalance{Username: "nobody"})
	if !errors.Is(err, model.ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

func TestGetBalanceReturnsCopy(t *testing.T) {
	h, store := newHandler()

	w, _ := h.GetBalance(GetBalance{Username: "alice"})
	w.Balance = 0

	store.View(func(users map[string]*model.Wallet) error {
		if users["alice"].Balance != 42 {
			t.Errorf("stored balance changed to %v", users["alice"].Balance)
		}
		return nil
	})
}
