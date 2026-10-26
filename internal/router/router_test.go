package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"api/internal/controller"
	"api/internal/repository"
	"api/internal/service/command"
	"api/internal/service/query"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestRoutes(t *testing.T) {
	store := repository.NewMemoryStore()
	r := New(controller.NewWalletController(command.NewHandler(store), query.NewHandler(store)))

	steps := []struct {
		method, path, body string
		wantStatus         int
		wantBody           string
	}{
		{"GET", "/ping", "", 200, `{"message":"pong"}`},
		{"POST", "/create", `{"username":"alice","balance":100}`, 201, `{"message":"Username alice created with balance 100.00"}`},
		{"POST", "/create", `{"username":"bob","balance":1}`, 201, `{"message":"Username bob created with balance 1.00"}`},
		{"POST", "/create", `{"username":"bob","balance":1}`, 400, `{"error":"User already exists"}`},
		{"POST", "/transfer", `{"from":"alice","to":"bob","amount":40}`, 200, `{"message":"Transfer Amount 40.00 success from alice to bob"}`},
		{"POST", "/transfer", `{"from":"alice","to":"bob","amount":61}`, 400, `{"error":"Insufficient fund"}`},
		{"POST", "/transfer", `{"from":"zed","to":"bob","amount":1}`, 404, `{"error":"Sender user not found"}`},
		{"POST", "/transfer", `{"from":"alice","to":"zed","amount":1}`, 404, `{"error":"Recipient user not found"}`},
		{"GET", "/getBalance/alice", "", 200, `{"balance":60}`},
		{"GET", "/getBalance/bob", "", 200, `{"balance":41}`},
		{"GET", "/getBalance/zed", "", 404, `{"error":"User not found"}`},
		{"GET", "/nope", "", 404, ""},
	}

	for _, s := range steps {
		req := httptest.NewRequest(s.method, s.path, strings.NewReader(s.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != s.wantStatus {
			t.Errorf("%s %s: status = %d, want %d", s.method, s.path, rec.Code, s.wantStatus)
			continue
		}
		if s.wantBody != "" && !sameJSON(t, rec.Body.String(), s.wantBody) {
			t.Errorf("%s %s: body = %s, want %s", s.method, s.path, rec.Body, s.wantBody)
		}
	}
}

func sameJSON(t *testing.T, a, b string) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal([]byte(a), &x); err != nil {
		t.Fatalf("bad JSON %q", a)
	}
	json.Unmarshal([]byte(b), &y)
	xb, _ := json.Marshal(x)
	yb, _ := json.Marshal(y)
	return string(xb) == string(yb)
}

func TestMethodsAreRestricted(t *testing.T) {
	store := repository.NewMemoryStore()
	r := New(controller.NewWalletController(command.NewHandler(store), query.NewHandler(store)))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/create", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /create status = %d, want 404", rec.Code)
	}
}

func TestEveryRouteIsLogged(t *testing.T) {
	hook := test.NewGlobal()
	defer hook.Reset()

	store := repository.NewMemoryStore()
	r := New(controller.NewWalletController(command.NewHandler(store), query.NewHandler(store)))

	steps := []struct {
		method, path, body string
		wantLevel          log.Level
	}{
		{"GET", "/ping", "", log.InfoLevel},
		{"POST", "/create", `{"username":"alice","balance":5}`, log.InfoLevel},
		{"GET", "/getBalance/alice", "", log.InfoLevel},
		{"POST", "/transfer", `{"from":"alice","to":"zed","amount":1}`, log.WarnLevel},
		{"GET", "/getBalance/zed", "", log.WarnLevel},
	}

	for _, s := range steps {
		hook.Reset()
		req := httptest.NewRequest(s.method, s.path, strings.NewReader(s.body))
		r.ServeHTTP(httptest.NewRecorder(), req)

		var access *log.Entry
		for _, e := range hook.AllEntries() {
			if _, ok := e.Data["status"]; ok {
				access = e
			}
		}
		if access == nil {
			t.Errorf("%s %s: no access log line", s.method, s.path)
			continue
		}
		if access.Level != s.wantLevel {
			t.Errorf("%s %s: level = %v, want %v", s.method, s.path, access.Level, s.wantLevel)
		}
	}
}
