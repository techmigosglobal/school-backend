package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"
	"school-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var JWTSecret []byte
var allowedOrigins = map[string]struct{}{}

func SetJWTSecret(secret string) {
	JWTSecret = []byte(secret)
}

func SetAllowedOrigins(origins []string) {
	allowedOrigins = map[string]struct{}{}
	for _, origin := range origins {
		clean := strings.TrimSpace(origin)
		if clean == "" {
			continue
		}
		allowedOrigins[clean] = struct{}{}
	}
}

type Claims struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	RoleID     string `json:"role_id"`
	RoleName   string `json:"role_name"`
	SchoolID   string `json:"school_id"`
	LinkedType string `json:"linked_type"`
	LinkedID   string `json:"linked_id"`
	JTI        string `json:"jti"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, email, roleID, roleName, schoolID, linkedType, linkedID, jti string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:     userID,
		Email:      email,
		RoleID:     roleID,
		RoleName:   roleName,
		SchoolID:   schoolID,
		LinkedType: linkedType,
		LinkedID:   linkedID,
		JTI:        jti,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "school-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondError(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			respondError(c, http.StatusUnauthorized, "Bearer token required")
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			respondError(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}
		if claims.JTI != "" && services.Sessions != nil {
			revoked, err := services.Sessions.IsJTIRevoked(context.Background(), claims.JTI)
			if err != nil {
				respondError(c, http.StatusUnauthorized, "Session validation failed")
				c.Abort()
				return
			}
			if revoked {
				respondError(c, http.StatusUnauthorized, "Token has been revoked")
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role_id", claims.RoleID)
		c.Set("role_name", claims.RoleName)
		c.Set("school_id", claims.SchoolID)
		c.Set("linked_type", claims.LinkedType)
		c.Set("linked_id", claims.LinkedID)
		c.Set("jti", claims.JTI)

		c.Next()
	}
}

func RBACMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleName, exists := c.Get("role_name")
		if !exists {
			respondError(c, http.StatusUnauthorized, "Role not found")
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if role == roleName.(string) {
				c.Next()
				return
			}
		}

		respondError(c, http.StatusForbidden, "Access denied")
		c.Abort()
	}
}

func PermissionMiddleware(module, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID := strings.TrimSpace(c.GetString("role_id"))
		roleName := c.GetString("role_name")
		if roleID == "" {
			respondError(c, http.StatusUnauthorized, "Role not found")
			c.Abort()
			return
		}
		if roleName == "Admin" {
			c.Next()
			return
		}

		var permission models.Permission
		err := database.DB.Where("role_id = ? AND module = ?", roleID, module).First(&permission).Error
		if err != nil {
			respondError(c, http.StatusForbidden, "Access denied")
			c.Abort()
			return
		}

		allowed := false
		switch strings.ToLower(action) {
		case "read":
			allowed = permission.CanRead
		case "create":
			allowed = permission.CanCreate
		case "update":
			allowed = permission.CanUpdate
		case "delete":
			allowed = permission.CanDelete
		case "export":
			allowed = permission.CanExport
		}
		if !allowed {
			respondError(c, http.StatusForbidden, "Access denied")
			c.Abort()
			return
		}
		c.Next()
	}
}

func SchoolScopeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenSchoolID := strings.TrimSpace(c.GetString("school_id"))
		if tokenSchoolID == "" {
			respondError(c, http.StatusForbidden, "school scope missing in token")
			c.Abort()
			return
		}

		requestSchoolID := strings.TrimSpace(c.Query("school_id"))
		if requestSchoolID != "" && requestSchoolID != tokenSchoolID {
			respondError(c, http.StatusForbidden, "cross-school access denied")
			c.Abort()
			return
		}

		// Force all handlers to use token school scope even if query param is omitted.
		query := c.Request.URL.Query()
		query.Set("school_id", tokenSchoolID)
		c.Request.URL.RawQuery = query.Encode()

		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && isOriginAllowed(origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func isOriginAllowed(origin string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}
	if _, ok := allowedOrigins[origin]; ok {
		return true
	}
	// Normalize to avoid trailing slash mismatch
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	normalized := strings.TrimSuffix(parsed.Scheme+"://"+parsed.Host, "/")
	_, ok := allowedOrigins[normalized]
	return ok
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"success":    false,
		"code":       http.StatusText(status),
		"message":    message,
		"error":      message,
		"request_id": c.GetString("request_id"),
	})
}
