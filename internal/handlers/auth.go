package handlers

import (
	"net/http"
	"school-backend/internal/database"
	"school-backend/internal/middleware"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
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

	token, err := middleware.GenerateToken(
		user.ID,
		user.Email,
		user.RoleID,
		user.Role.RoleName,
		user.SchoolID,
		user.LinkedType,
		"",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: models.LoginResponse{
			Token:        token,
			RefreshToken: "",
			ExpiresAt:    86400,
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

	user := models.User{
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hashedPassword,
		SchoolID:     req.SchoolID,
		RoleID:       req.RoleID,
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