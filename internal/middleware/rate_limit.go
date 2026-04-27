package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"school-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func RateLimitMiddleware(endpoint string, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if services.Rate == nil {
			c.Next()
			return
		}
		subject := c.ClientIP()
		if uid := c.GetString("user_id"); uid != "" {
			subject = uid
		}
		allowed, remaining, err := services.Rate.Allow(context.Background(), endpoint, subject, maxRequests, window)
		if err != nil {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limiter unavailable"})
			c.Abort()
			return
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}
