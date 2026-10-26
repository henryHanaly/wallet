package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"

	"api/internal/model"
	"api/internal/service/command"
	"api/internal/service/query"
)

type CommandService interface {
	CreateUser(cmd command.CreateUser) error
	Transfer(cmd command.Transfer) error
}

type QueryService interface {
	GetBalance(q query.GetBalance) (model.Wallet, error)
}

type createUserRequest struct {
	Username string  `json:"username" validate:"required,min=3,max=20"`
	Balance  float32 `json:"balance" validate:"gt=0"`
}

type transferRequest struct {
	From   string  `json:"from" validate:"required"`
	To     string  `json:"to" validate:"required"`
	Amount float32 `json:"amount" validate:"gt=0"`
}

type WalletController struct {
	commands CommandService
	queries  QueryService
	validate *validator.Validate
}

func NewWalletController(commands CommandService, queries QueryService) *WalletController {
	return &WalletController{
		commands: commands,
		queries:  queries,
		validate: validator.New(),
	}
}

func (w *WalletController) CreateUser(c *gin.Context) {
	var req createUserRequest
	if !w.bind(c, &req) {
		return
	}

	err := w.commands.CreateUser(command.CreateUser{
		Username: req.Username,
		Balance:  req.Balance,
	})
	if err != nil {
		log.WithField("username", req.Username).Error(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("Username %s created with balance %.2f", req.Username, req.Balance),
	})
}

func (w *WalletController) GetBalance(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))

	if username == "" || username == ":username" {
		log.WithField("handler", "getBalance").Error("Username cannot be empty")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username cannot be empty"})
		return
	}

	wallet, err := w.queries.GetBalance(query.GetBalance{Username: username})
	if err != nil {
		log.WithFields(log.Fields{
			"handler":  "getBalance",
			"username": username,
		}).Error(err)
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (w *WalletController) Transfer(c *gin.Context) {
	var req transferRequest
	if !w.bind(c, &req) {
		return
	}

	err := w.commands.Transfer(command.Transfer{
		From:   req.From,
		To:     req.To,
		Amount: req.Amount,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"from":   req.From,
			"to":     req.To,
			"amount": req.Amount,
		}).Error(err)
		writeError(c, err)
		return
	}

	msg := fmt.Sprintf("Transfer Amount %.2f success from %s to %s", req.Amount, req.From, req.To)
	log.WithFields(log.Fields{
		"from":   req.From,
		"to":     req.To,
		"amount": req.Amount,
	}).Info(msg)

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

func (w *WalletController) bind(c *gin.Context, req any) bool {
	err := c.ShouldBindJSON(req)
	if err == nil {
		err = w.validate.Struct(req)
	}
	if err != nil {
		log.Error(fmt.Sprintf("error: %s", err.Error()))
		c.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func writeError(c *gin.Context, err error) {
	c.Error(err)
	status, msg := http.StatusInternalServerError, "Internal server error"

	switch {
	case errors.Is(err, model.ErrUserExists):
		status, msg = http.StatusBadRequest, "User already exists"
	case errors.Is(err, model.ErrUserNotFound):
		status, msg = http.StatusNotFound, "User not found"
	case errors.Is(err, model.ErrSenderNotFound):
		status, msg = http.StatusNotFound, "Sender user not found"
	case errors.Is(err, model.ErrRecipientNotFound):
		status, msg = http.StatusNotFound, "Recipient user not found"
	case errors.Is(err, model.ErrInsufficientFunds):
		status, msg = http.StatusBadRequest, "Insufficient fund"
	case errors.Is(err, model.ErrInvalidAmount):
		status, msg = http.StatusBadRequest, "Amount must be greater than zero"
	}

	c.JSON(status, gin.H{"error": msg})
}
