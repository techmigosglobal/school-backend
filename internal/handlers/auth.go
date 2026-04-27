package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/middleware"
	"school-backend/internal/models"
	"school-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Preload("Role").Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !database.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is deactivated"})
		return
	}

	jti := uuid.NewString()
	tokenTTL := 24 * time.Hour
	token, err := middleware.GenerateToken(
		user.ID,
		user.Email,
		user.RoleID,
		user.Role.RoleName,
		user.SchoolID,
		user.LinkedType,
		"",
		jti,
		tokenTTL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken := uuid.NewString()
	refreshPayload, _ := json.Marshal(map[string]string{
		"user_id":     user.ID,
		"email":       user.Email,
		"role_id":     user.RoleID,
		"role_name":   user.Role.RoleName,
		"school_id":   user.SchoolID,
		"linked_type": user.LinkedType,
	})
	if services.Sessions != nil {
		if err := services.Sessions.StoreRefreshToken(context.Background(), refreshToken, string(refreshPayload), 7*24*time.Hour); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist session"})
			return
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: models.LoginResponse{
			Token:        token,
			RefreshToken: refreshToken,
			ExpiresAt:    int64(tokenTTL.Seconds()),
			User: models.UserResponse{
				ID:         user.ID,
				Email:      user.Email,
				Phone:      user.Phone,
				SchoolID:   user.SchoolID,
				RoleID:     user.RoleID,
				RoleName:   user.Role.RoleName,
				IsActive:   user.IsActive,
				IsVerified: user.IsVerified,
			},
		},
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := database.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if strings.TrimSpace(req.RoleID) != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_id cannot be provided for public registration"})
		return
	}

	var parentRole models.Role
	if err := database.DB.Where("LOWER(role_name) = ?", "parent").First(&parentRole).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Default role not available"})
		return
	}

	user := models.User{
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hashedPassword,
		SchoolID:     req.SchoolID,
		RoleID:       parentRole.ID,
		IsActive:     true,
		IsVerified:   false,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "User registered successfully",
		Data:    user.ID,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if services.Sessions == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session store not available"})
		return
	}

	rawPayload, err := services.Sessions.GetRefreshToken(context.Background(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token payload"})
		return
	}

	_ = services.Sessions.RevokeRefreshToken(context.Background(), req.RefreshToken)
	newRefresh := uuid.NewString()
	if err := services.Sessions.StoreRefreshToken(context.Background(), newRefresh, rawPayload, 7*24*time.Hour); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rotate refresh token"})
		return
	}

	jti := uuid.NewString()
	accessTTL := 24 * time.Hour
	token, err := middleware.GenerateToken(
		payload["user_id"],
		payload["email"],
		payload["role_id"],
		payload["role_name"],
		payload["school_id"],
		payload["linked_type"],
		"",
		jti,
		accessTTL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh access token"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: gin.H{
			"token":         token,
			"refresh_token": newRefresh,
			"expires_at":    int64(accessTTL.Seconds()),
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&req)

	if services.Sessions != nil {
		jti := c.GetString("jti")
		if jti != "" {
			_ = services.Sessions.RevokeJTI(context.Background(), jti, 24*time.Hour)
		}
		if strings.TrimSpace(req.RefreshToken) != "" {
			_ = services.Sessions.RevokeRefreshToken(context.Background(), req.RefreshToken)
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var user models.User
	if err := database.DB.Preload("Role").Preload("Role.Permissions").First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: models.UserResponse{
			ID:         user.ID,
			Email:      user.Email,
			Phone:      user.Phone,
			SchoolID:   user.SchoolID,
			RoleID:     user.RoleID,
			RoleName:   user.Role.RoleName,
			IsActive:   user.IsActive,
			IsVerified: user.IsVerified,
		},
	})
}
