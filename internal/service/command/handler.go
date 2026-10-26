package command

import (
	"api/internal/model"
	"api/internal/repository"
)

type CreateUser struct {
	Username string
	Balance  float32
}

type Transfer struct {
	From   string
	To     string
	Amount float32
}

type Handler struct {
	store repository.Store
}

func NewHandler(store repository.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) CreateUser(cmd CreateUser) error {
	return h.store.Update(func(users map[string]*model.Wallet) error {
		if _, present := users[cmd.Username]; present {
			return model.ErrUserExists
		}
		users[cmd.Username] = &model.Wallet{Balance: cmd.Balance}
		return nil
	})
}

func (h *Handler) Transfer(cmd Transfer) error {
	if cmd.Amount <= 0 {
		return model.ErrInvalidAmount
	}

	return h.store.Update(func(users map[string]*model.Wallet) error {
		from, present := users[cmd.From]
		if !present {
			return model.ErrSenderNotFound
		}

		to, present := users[cmd.To]
		if !present {
			return model.ErrRecipientNotFound
		}

		if from.Balance < cmd.Amount {
			return model.ErrInsufficientFunds
		}

		from.Balance -= cmd.Amount
		to.Balance += cmd.Amount
		return nil
	})
}
