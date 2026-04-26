package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var JWTSecret = []byte("school-desk-secret-key-2024")

func SetJWTSecret(secret string) {
	if strings.TrimSpace(secret) == "" {
		return
	}
	JWTSecret = []byte(secret)
}

type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	RoleID    string `json:"role_id"`
	RoleName  string `json:"role_name"`
	SchoolID  string `json:"school_id"`
	LinkedType string `json:"linked_type"`
	LinkedID  string `json:"linked_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, email, roleID, roleName, schoolID, linkedType, linkedID string) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		RoleID:    roleID,
		RoleName:  roleName,
		SchoolID:  schoolID,
		LinkedType: linkedType,
		LinkedID:  linkedID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role_id", claims.RoleID)
		c.Set("role_name", claims.RoleName)
		c.Set("school_id", claims.SchoolID)
		c.Set("linked_type", claims.LinkedType)
		c.Set("linked_id", claims.LinkedID)

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleName, exists := c.Get("role_name")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Role not found"})
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if role == roleName.(string) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		c.Abort()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
