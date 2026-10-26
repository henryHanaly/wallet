package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func Logger(logger *log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := c.Writer.Status()
		entry := logger.WithFields(log.Fields{
			"status":    status,
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"client_ip": c.ClientIP(),
			"latency":   time.Since(start).String(),
		})
		if len(c.Errors) > 0 {
			entry = entry.WithField("errors", c.Errors.String())
		}

		msg := http.StatusText(status)
		switch {
		case status >= http.StatusInternalServerError:
			entry.Error(msg)
		case status >= http.StatusBadRequest:
			entry.Warn(msg)
		default:
			entry.Info(msg)
		}
	}
}
