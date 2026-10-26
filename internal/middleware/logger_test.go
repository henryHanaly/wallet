package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func newEngine(logger *log.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Logger(logger), gin.Recovery())

	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/created", func(c *gin.Context) { c.Status(http.StatusCreated) })
	r.GET("/bad", func(c *gin.Context) {
		c.Error(errors.New("username too short"))
		c.Status(http.StatusBadRequest)
	})
	r.GET("/fail", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })
	r.GET("/panic", func(c *gin.Context) { panic("boom") })
	return r
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		method, path string
		wantStatus   int
		wantLevel    log.Level
		wantMsg      string
	}{
		{"GET", "/ok", 200, log.InfoLevel, "OK"},
		{"POST", "/created", 201, log.InfoLevel, "Created"},
		{"GET", "/bad", 400, log.WarnLevel, "Bad Request"},
		{"GET", "/missing", 404, log.WarnLevel, "Not Found"},
		{"GET", "/fail", 500, log.ErrorLevel, "Internal Server Error"},
		{"GET", "/panic", 500, log.ErrorLevel, "Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			logger, hook := test.NewNullLogger()
			r := newEngine(logger)
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tt.method, tt.path, nil))

			if len(hook.Entries) != 1 {
				t.Fatalf("got %d log entries, want 1", len(hook.Entries))
			}
			e := hook.LastEntry()
			if e.Level != tt.wantLevel {
				t.Errorf("level = %v, want %v", e.Level, tt.wantLevel)
			}
			if e.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", e.Message, tt.wantMsg)
			}
			if e.Data["status"] != tt.wantStatus {
				t.Errorf("status field = %v, want %d", e.Data["status"], tt.wantStatus)
			}
			if e.Data["method"] != tt.method || e.Data["path"] != tt.path {
				t.Errorf("method/path = %v %v", e.Data["method"], e.Data["path"])
			}
			for _, k := range []string{"client_ip", "latency"} {
				if _, ok := e.Data[k]; !ok {
					t.Errorf("missing field %q", k)
				}
			}
		})
	}
}

func TestLoggerIncludesHandlerErrors(t *testing.T) {
	logger, hook := test.NewNullLogger()
	r := newEngine(logger)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/bad", nil))

	got, _ := hook.LastEntry().Data["errors"].(string)
	if got == "" {
		t.Fatal("errors field missing")
	}
	if want := "username too short"; !strings.Contains(got, want) {
		t.Errorf("errors = %q, want it to mention %q", got, want)
	}

	hook.Reset()
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/ok", nil))
	if _, ok := hook.LastEntry().Data["errors"]; ok {
		t.Error("errors field should be absent on success")
	}
}
