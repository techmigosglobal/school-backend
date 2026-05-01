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
	"gorm.io/gorm"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func buildRefreshPayload(user models.User, roleName string) string {
	refreshPayload, _ := json.Marshal(map[string]string{
		"user_id":     user.ID,
		"email":       user.Email,
		"role_id":     user.RoleID,
		"role_name":   roleName,
		"school_id":   user.SchoolID,
		"linked_type": user.LinkedType,
		"linked_id":   stringValue(user.LinkedID),
	})
	return string(refreshPayload)
}

func adminNameParts(name string) (string, string) {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "School", "Admin"
	}
	if len(parts) == 1 {
		return parts[0], "Admin"
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func adminStaffCode(schoolID string) string {
	compact := strings.ReplaceAll(schoolID, "-", "")
	if len(compact) > 8 {
		compact = compact[:8]
	}
	return "ADM-" + strings.ToUpper(compact)
}

func issueLoginResponse(c *gin.Context, user models.User, roleName string) {
	jti := uuid.NewString()
	tokenTTL := 24 * time.Hour
	token, err := middleware.GenerateToken(
		user.ID,
		user.Email,
		user.RoleID,
		roleName,
		user.SchoolID,
		user.LinkedType,
		stringValue(user.LinkedID),
		jti,
		tokenTTL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken := uuid.NewString()
	if services.Sessions != nil {
		if err := services.Sessions.StoreRefreshToken(context.Background(), refreshToken, buildRefreshPayload(user, roleName), 7*24*time.Hour); err != nil {
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
				Username:   user.Username,
				Email:      user.Email,
				Phone:      user.Phone,
				SchoolID:   user.SchoolID,
				RoleID:     user.RoleID,
				RoleName:   roleName,
				IsActive:   user.IsActive,
				IsVerified: user.IsVerified,
			},
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	identifier := strings.ToLower(strings.TrimSpace(req.Username))
	if err := database.DB.Preload("Role").
		Where("LOWER(username) = ? OR LOWER(email) = ?", identifier, identifier).
		First(&user).Error; err != nil {
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

	issueLoginResponse(c, user, user.Role.RoleName)
}

func (h *AuthHandler) RegisterSchoolAdmin(c *gin.Context) {
	var req models.RegisterSchoolAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.AdminEmail))
	var existing int64
	database.DB.Model(&models.User{}).Where("LOWER(email) = ?", email).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin email is already registered"})
		return
	}

	hashedPassword, err := database.HashPassword(req.AdminPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var createdUser models.User
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		schoolType := strings.TrimSpace(req.SchoolType)
		if schoolType == "" {
			schoolType = "K-12"
		}
		timezone := strings.TrimSpace(req.Timezone)
		if timezone == "" {
			timezone = "Asia/Kolkata"
		}
		currency := strings.TrimSpace(req.Currency)
		if currency == "" {
			currency = "INR"
		}

		school := models.School{
			Name:             strings.TrimSpace(req.SchoolName),
			SchoolType:       schoolType,
			AffiliationBoard: strings.TrimSpace(req.AffiliationBoard),
			Email:            strings.TrimSpace(req.SchoolEmail),
			Phone:            strings.TrimSpace(req.SchoolPhone),
			City:             strings.TrimSpace(req.City),
			State:            strings.TrimSpace(req.State),
			Timezone:         timezone,
			Currency:         currency,
		}
		if err := tx.Create(&school).Error; err != nil {
			return err
		}

		roleIDs := map[string]string{}
		for _, roleName := range []string{"Admin", "Principal", "Teacher", "Parent"} {
			role := models.Role{
				SchoolID:     school.ID,
				RoleName:     roleName,
				Description:  roleName + " role",
				IsSystemRole: true,
			}
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
			roleIDs[strings.ToLower(roleName)] = role.ID
		}
		bootstrapRolePermissions(tx, roleIDs)

		firstName, lastName := adminNameParts(req.AdminName)
		staff := models.Staff{
			SchoolID:       school.ID,
			StaffCode:      adminStaffCode(school.ID),
			FirstName:      firstName,
			LastName:       lastName,
			Email:          email,
			Phone:          strings.TrimSpace(req.AdminPhone),
			Designation:    "Administrator",
			EmploymentType: "permanent",
			JoinDate:       time.Now().UTC(),
			Status:         "active",
		}
		if err := tx.Create(&staff).Error; err != nil {
			return err
		}
		staffID := staff.ID

		createdUser = models.User{
			Username:     strings.ToUpper(strings.TrimSpace(req.AdminEmail)),
			Email:        email,
			Phone:        strings.TrimSpace(req.AdminPhone),
			PasswordHash: hashedPassword,
			SchoolID:     school.ID,
			RoleID:       roleIDs["admin"],
			LinkedType:   "staff",
			LinkedID:     &staffID,
			IsActive:     true,
			IsVerified:   true,
		}
		if err := tx.Create(&createdUser).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register school admin"})
		return
	}

	issueLoginResponse(c, createdUser, "Admin")
}

func bootstrapRolePermissions(tx *gorm.DB, roleIDs map[string]string) {
	modules := []string{
		"dashboard",
		"guardians",
		"medical_records",
		"student_documents",
		"staff_documents",
		"staff_subjects",
		"staff_qualifications",
		"transport",
		"library",
		"payroll",
		"parent_teacher_meetings",
		"homework",
		"diary_entries",
		"message_conversations",
		"messages",
		"audit_logs",
	}
	contains := func(value string, items ...string) bool {
		for _, item := range items {
			if value == item {
				return true
			}
		}
		return false
	}
	upsert := func(roleID, module string, read, create, update, delete, export bool) {
		if roleID == "" {
			return
		}
		values := models.Permission{
			RoleID:    roleID,
			Module:    module,
			CanRead:   read,
			CanCreate: create,
			CanUpdate: update,
			CanDelete: delete,
			CanExport: export,
		}
		tx.Where("role_id = ? AND module = ?", roleID, module).
			Assign(values).
			FirstOrCreate(&models.Permission{})
	}

	for _, module := range modules {
		upsert(roleIDs["admin"], module, true, true, true, true, true)
		upsert(roleIDs["principal"], module, true, true, true, module != "audit_logs", true)

		teacherRead := contains(module, "dashboard", "guardians", "medical_records", "student_documents", "staff_subjects", "staff_qualifications", "library", "parent_teacher_meetings", "homework", "diary_entries", "message_conversations", "messages")
		teacherManage := contains(module, "homework", "diary_entries", "message_conversations", "messages", "parent_teacher_meetings")
		upsert(roleIDs["teacher"], module, teacherRead, teacherManage, teacherManage, false, false)

		parentRead := contains(module, "dashboard", "guardians", "medical_records", "student_documents", "parent_teacher_meetings", "homework", "diary_entries", "message_conversations", "messages")
		parentCreate := contains(module, "parent_teacher_meetings", "message_conversations", "messages")
		parentUpdate := contains(module, "message_conversations", "messages")
		upsert(roleIDs["parent"], module, parentRead, parentCreate, parentUpdate, false, false)
	}
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
