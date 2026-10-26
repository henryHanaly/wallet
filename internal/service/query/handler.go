package query

import (
	"api/internal/model"
	"api/internal/repository"
)

type GetBalance struct {
	Username string
}

type Handler struct {
	store repository.Store
}

func NewHandler(store repository.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) GetBalance(q GetBalance) (model.Wallet, error) {
	var wallet model.Wallet
	err := h.store.View(func(users map[string]*model.Wallet) error {
		user, present := users[q.Username]
		if !present {
			return model.ErrUserNotFound
		}
		wallet = *user
		return nil
	})
	return wallet, err
}
