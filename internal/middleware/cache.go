package middleware

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"school-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type cachedResponseWriter struct {
	gin.ResponseWriter
	body   []byte
	status int
}

func (w *cachedResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *cachedResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func CacheMiddleware(routeName string, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if services.Cache == nil || c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		sum := sha1.Sum([]byte(c.Request.URL.String()))
		key := routeName + ":" + hex.EncodeToString(sum[:])
		if cached, err := services.Cache.Get(context.Background(), key); err == nil && cached != "" {
			var payload interface{}
			if err := json.Unmarshal([]byte(cached), &payload); err == nil {
				c.JSON(http.StatusOK, payload)
				c.Abort()
				return
			}
		}

		writer := &cachedResponseWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = writer
		c.Next()

		if writer.status >= 200 && writer.status < 300 && len(writer.body) > 0 {
			_ = services.Cache.Set(context.Background(), key, string(writer.body), ttl)
		}
	}
}
