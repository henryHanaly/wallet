package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"api/internal/model"
	"api/internal/service/command"
	"api/internal/service/query"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

type fakeCommands struct {
	err         error
	gotCreate   command.CreateUser
	gotTransfer command.Transfer
}

func (f *fakeCommands) CreateUser(cmd command.CreateUser) error {
	f.gotCreate = cmd
	return f.err
}

func (f *fakeCommands) Transfer(cmd command.Transfer) error {
	f.gotTransfer = cmd
	return f.err
}

type fakeQueries struct {
	wallet model.Wallet
	err    error
	got    query.GetBalance
}

func (f *fakeQueries) GetBalance(q query.GetBalance) (model.Wallet, error) {
	f.got = q
	return f.wallet, f.err
}

func call(handler gin.HandlerFunc, method, body string, params gin.Params) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	handler(c)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad JSON %q: %v", rec.Body.String(), err)
	}
	return out
}

func TestPing(t *testing.T) {
	rec := call(Ping, http.MethodGet, "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decode(t, rec)["message"]; got != "pong" {
		t.Errorf("message = %v", got)
	}
}

func TestCreateUser(t *testing.T) {
	cmds := &fakeCommands{}
	w := NewWalletController(cmds, &fakeQueries{})

	rec := call(w.CreateUser, http.MethodPost, `{"username":"alice","balance":12.5}`, nil)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if got := decode(t, rec)["message"]; got != "Username alice created with balance 12.50" {
		t.Errorf("message = %v", got)
	}
	if cmds.gotCreate != (command.CreateUser{Username: "alice", Balance: 12.5}) {
		t.Errorf("command = %+v", cmds.gotCreate)
	}
}

func TestCreateUserBadInput(t *testing.T) {
	bodies := map[string]string{
		"not json":       `{`,
		"wrong type":     `{"username":"alice","balance":"ten"}`,
		"missing name":   `{"balance":10}`,
		"short name":     `{"username":"al","balance":10}`,
		"long name":      `{"username":"abcdefghijklmnopqrstu","balance":10}`,
		"zero balance":   `{"username":"alice","balance":0}`,
		"negative funds": `{"username":"alice","balance":-1}`,
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			cmds := &fakeCommands{}
			w := NewWalletController(cmds, &fakeQueries{})

			rec := call(w.CreateUser, http.MethodPost, body, nil)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d", rec.Code)
			}
			if decode(t, rec)["error"] == nil {
				t.Error("no error message in body")
			}
			if cmds.gotCreate != (command.CreateUser{}) {
				t.Error("service should not be called on bad input")
			}
		})
	}
}

func TestCreateUserServiceError(t *testing.T) {
	w := NewWalletController(&fakeCommands{err: model.ErrUserExists}, &fakeQueries{})

	rec := call(w.CreateUser, http.MethodPost, `{"username":"alice","balance":1}`, nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decode(t, rec)["error"]; got != "User already exists" {
		t.Errorf("error = %v", got)
	}
}

func TestGetBalance(t *testing.T) {
	qs := &fakeQueries{wallet: model.Wallet{Balance: 7.5}}
	w := NewWalletController(&fakeCommands{}, qs)

	rec := call(w.GetBalance, http.MethodGet, "", gin.Params{{Key: "username", Value: "  alice "}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decode(t, rec)["balance"]; got != 7.5 {
		t.Errorf("balance = %v", got)
	}
	if qs.got.Username != "alice" {
		t.Errorf("username not trimmed: %q", qs.got.Username)
	}
}

func TestGetBalanceEmptyUsername(t *testing.T) {
	for _, value := range []string{"", "   ", ":username"} {
		t.Run(value, func(t *testing.T) {
			w := NewWalletController(&fakeCommands{}, &fakeQueries{})

			rec := call(w.GetBalance, http.MethodGet, "", gin.Params{{Key: "username", Value: value}})

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d", rec.Code)
			}
			if got := decode(t, rec)["error"]; got != "Username cannot be empty" {
				t.Errorf("error = %v", got)
			}
		})
	}
}

func TestGetBalanceNotFound(t *testing.T) {
	w := NewWalletController(&fakeCommands{}, &fakeQueries{err: model.ErrUserNotFound})

	rec := call(w.GetBalance, http.MethodGet, "", gin.Params{{Key: "username", Value: "ghost"}})

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decode(t, rec)["error"]; got != "User not found" {
		t.Errorf("error = %v", got)
	}
}

func TestTransfer(t *testing.T) {
	cmds := &fakeCommands{}
	w := NewWalletController(cmds, &fakeQueries{})

	rec := call(w.Transfer, http.MethodPost, `{"from":"alice","to":"bob","amount":5}`, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if got := decode(t, rec)["message"]; got != "Transfer Amount 5.00 success from alice to bob" {
		t.Errorf("message = %v", got)
	}
	if cmds.gotTransfer != (command.Transfer{From: "alice", To: "bob", Amount: 5}) {
		t.Errorf("command = %+v", cmds.gotTransfer)
	}
}

func TestTransferBadInput(t *testing.T) {
	bodies := map[string]string{
		"not json":     `[`,
		"missing from": `{"to":"bob","amount":5}`,
		"missing to":   `{"from":"alice","amount":5}`,
		"zero amount":  `{"from":"alice","to":"bob","amount":0}`,
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			cmds := &fakeCommands{}
			w := NewWalletController(cmds, &fakeQueries{})

			rec := call(w.Transfer, http.MethodPost, body, nil)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d", rec.Code)
			}
			if cmds.gotTransfer != (command.Transfer{}) {
				t.Error("service should not be called on bad input")
			}
		})
	}
}

func TestTransferServiceError(t *testing.T) {
	w := NewWalletController(&fakeCommands{err: model.ErrInsufficientFunds}, &fakeQueries{})

	rec := call(w.Transfer, http.MethodPost, `{"from":"alice","to":"bob","amount":5}`, nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decode(t, rec)["error"]; got != "Insufficient fund" {
		t.Errorf("error = %v", got)
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantMsg    string
	}{
		{model.ErrUserExists, http.StatusBadRequest, "User already exists"},
		{model.ErrUserNotFound, http.StatusNotFound, "User not found"},
		{model.ErrSenderNotFound, http.StatusNotFound, "Sender user not found"},
		{model.ErrRecipientNotFound, http.StatusNotFound, "Recipient user not found"},
		{model.ErrInsufficientFunds, http.StatusBadRequest, "Insufficient fund"},
		{model.ErrInvalidAmount, http.StatusBadRequest, "Amount must be greater than zero"},
		{errors.New("disk on fire"), http.StatusInternalServerError, "Internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			writeError(c, tt.err)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := decode(t, rec)["error"]; got != tt.wantMsg {
				t.Errorf("error = %v, want %q", got, tt.wantMsg)
			}
		})
	}
}
