package command

import (
	"errors"
	"testing"

	"api/internal/model"
	"api/internal/repository"
)

func newHandler(t *testing.T, seed map[string]float32) (*Handler, *repository.MemoryStore) {
	t.Helper()
	store := repository.NewMemoryStore()
	store.Update(func(users map[string]*model.Wallet) error {
		for name, balance := range seed {
			users[name] = &model.Wallet{Balance: balance}
		}
		return nil
	})
	return NewHandler(store), store
}

func balanceOf(t *testing.T, store *repository.MemoryStore, name string) float32 {
	t.Helper()
	var b float32
	store.View(func(users map[string]*model.Wallet) error {
		w, ok := users[name]
		if !ok {
			t.Fatalf("user %q missing from store", name)
		}
		b = w.Balance
		return nil
	})
	return b
}

func TestCreateUser(t *testing.T) {
	h, store := newHandler(t, nil)

	if err := h.CreateUser(CreateUser{Username: "alice", Balance: 50}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if got := balanceOf(t, store, "alice"); got != 50 {
		t.Errorf("balance = %v, want 50", got)
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	h, store := newHandler(t, map[string]float32{"alice": 50})

	err := h.CreateUser(CreateUser{Username: "alice", Balance: 999})
	if !errors.Is(err, model.ErrUserExists) {
		t.Fatalf("err = %v, want ErrUserExists", err)
	}
	if got := balanceOf(t, store, "alice"); got != 50 {
		t.Errorf("existing balance changed to %v", got)
	}
}

func TestTransfer(t *testing.T) {
	tests := []struct {
		name     string
		cmd      Transfer
		wantErr  error
		wantFrom float32
		wantTo   float32
	}{
		{"moves money", Transfer{"alice", "bob", 30}, nil, 70, 30},
		{"whole balance", Transfer{"alice", "bob", 100}, nil, 0, 100},
		{"too much", Transfer{"alice", "bob", 100.01}, model.ErrInsufficientFunds, 100, 0},
		{"zero amount", Transfer{"alice", "bob", 0}, model.ErrInvalidAmount, 100, 0},
		{"negative amount", Transfer{"alice", "bob", -5}, model.ErrInvalidAmount, 100, 0},
		{"unknown sender", Transfer{"carol", "bob", 10}, model.ErrSenderNotFound, 100, 0},
		{"unknown recipient", Transfer{"alice", "carol", 10}, model.ErrRecipientNotFound, 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, store := newHandler(t, map[string]float32{"alice": 100, "bob": 0})

			err := h.Transfer(tt.cmd)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got := balanceOf(t, store, "alice"); got != tt.wantFrom {
				t.Errorf("alice = %v, want %v", got, tt.wantFrom)
			}
			if got := balanceOf(t, store, "bob"); got != tt.wantTo {
				t.Errorf("bob = %v, want %v", got, tt.wantTo)
			}
		})
	}
}

func TestTransferToSelf(t *testing.T) {
	h, store := newHandler(t, map[string]float32{"alice": 100})

	if err := h.Transfer(Transfer{"alice", "alice", 40}); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if got := balanceOf(t, store, "alice"); got != 100 {
		t.Errorf("balance = %v, want 100", got)
	}
}
