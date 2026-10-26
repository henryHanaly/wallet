package model

import "errors"

type Wallet struct {
	Balance float32 `json:"balance"`
}

var (
	ErrUserExists        = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrSenderNotFound    = errors.New("sender not found")
	ErrRecipientNotFound = errors.New("recipient not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
)
