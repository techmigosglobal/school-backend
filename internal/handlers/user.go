package handlers

import (
	"net/http"
	"strings"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

type userWriteRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Role     string `json:"role"`
	FullName string `json:"full_name"`
	IsActive *bool  `json:"is_active"`
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	page, pageSize := parsePagination(c)
	schoolID := scopedSchoolID(c)
	roleName := strings.TrimSpace(c.Query("role"))
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))

	var users []models.User
	var total int64

	query := database.DB.Model(&models.User{}).
		Where("school_id = ?", schoolID)

	if roleName != "" {
		query = query.Joins("JOIN roles ON roles.id = users.role_id").
			Where("LOWER(roles.role_name) = ?", strings.ToLower(roleName))
	}

	if status != "" {
		switch status {
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		}
	}

	query.Count(&total)
	if err := query.
		Preload("Role").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	result := make([]gin.H, 0, len(users))
	for _, u := range users {
		role := ""
		if u.Role != nil {
			role = u.Role.RoleName
		}
		result = append(result, gin.H{
			"id":          u.ID,
			"username":    u.Username,
			"email":       u.Email,
			"phone":       u.Phone,
			"school_id":   u.SchoolID,
			"role_id":     u.RoleID,
			"role_name":   role,
			"is_active":   u.IsActive,
			"is_verified": u.IsVerified,
			"last_login":  u.LastLogin,
			"created_at":  u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, paginationResult(page, pageSize, total, result))
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req userWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	roleName := normalizeRole(req.Role)
	if roleName == "" {
		fail(c, http.StatusBadRequest, "role is required")
		return
	}
	if !canManageUserRole(c, roleName) {
		fail(c, http.StatusForbidden, "You cannot create users for this role")
		return
	}

	username := strings.ToUpper(strings.TrimSpace(req.Username))
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if username == "" {
		username = strings.ToUpper(email)
	}
	if username == "" {
		fail(c, http.StatusBadRequest, "username is required")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		fail(c, http.StatusBadRequest, "password is required")
		return
	}
	if len(req.Password) < 6 {
		fail(c, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}
	if email == "" {
		email = strings.ToLower(username) + "@schooldesk.local"
	}

	schoolID := scopedSchoolID(c)
	var role models.Role
	if err := database.DB.Where("school_id = ? AND LOWER(role_name) = ?", schoolID, strings.ToLower(roleName)).First(&role).Error; err != nil {
		fail(c, http.StatusBadRequest, "role not found for this school")
		return
	}

	var existing int64
	database.DB.Model(&models.User{}).
		Where("school_id = ? AND (LOWER(username) = ? OR LOWER(email) = ?)", schoolID, strings.ToLower(username), strings.ToLower(email)).
		Count(&existing)
	if existing > 0 {
		fail(c, http.StatusConflict, "username or email already exists")
		return
	}

	hash, err := database.HashPassword(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	var created models.User
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		linkedType := ""
		var linkedID *string
		if roleName == "Admin" || roleName == "Teacher" {
			first, last := splitName(req.FullName, roleName)
			staff := models.Staff{
				SchoolID:       schoolID,
				StaffCode:      staffCode(roleName, username),
				FirstName:      first,
				LastName:       last,
				Email:          email,
				Phone:          strings.TrimSpace(req.Phone),
				Designation:    roleName,
				EmploymentType: "permanent",
				JoinDate:       time.Now().UTC(),
				Status:         "active",
			}
			if err := tx.Create(&staff).Error; err != nil {
				return err
			}
			linkedType = "staff"
			linkedID = &staff.ID
		}

		created = models.User{
			SchoolID:     schoolID,
			Username:     username,
			Email:        email,
			Phone:        strings.TrimSpace(req.Phone),
			PasswordHash: hash,
			RoleID:       role.ID,
			LinkedType:   linkedType,
			LinkedID:     linkedID,
			IsActive:     active,
			IsVerified:   true,
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to create user")
		return
	}

	auditAction(c, "users", "create", "users", &created.ID)
	success(c, http.StatusCreated, userPayload(created, roleName), "User created")
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req userWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	schoolID := scopedSchoolID(c)
	var user models.User
	if err := database.DB.Preload("Role").Where("id = ? AND school_id = ?", id, schoolID).First(&user).Error; err != nil {
		fail(c, http.StatusNotFound, "user not found")
		return
	}
	currentRoleName := ""
	if user.Role != nil {
		currentRoleName = user.Role.RoleName
	}
	if !canManageUserRole(c, currentRoleName) {
		fail(c, http.StatusForbidden, "You cannot update this user")
		return
	}

	nextRoleName := currentRoleName
	if strings.TrimSpace(req.Role) != "" {
		nextRoleName = normalizeRole(req.Role)
		if !canManageUserRole(c, nextRoleName) {
			fail(c, http.StatusForbidden, "You cannot assign this role")
			return
		}
		var role models.Role
		if err := database.DB.Where("school_id = ? AND LOWER(role_name) = ?", schoolID, strings.ToLower(nextRoleName)).First(&role).Error; err != nil {
			fail(c, http.StatusBadRequest, "role not found for this school")
			return
		}
		user.RoleID = role.ID
	}

	if username := strings.ToUpper(strings.TrimSpace(req.Username)); username != "" {
		user.Username = username
	}
	if email := strings.ToLower(strings.TrimSpace(req.Email)); email != "" {
		user.Email = email
	}
	if req.Phone != "" {
		user.Phone = strings.TrimSpace(req.Phone)
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if strings.TrimSpace(req.Password) != "" {
		if len(req.Password) < 6 {
			fail(c, http.StatusBadRequest, "password must be at least 6 characters")
			return
		}
		hash, err := database.HashPassword(req.Password)
		if err != nil {
			fail(c, http.StatusInternalServerError, "failed to hash password")
			return
		}
		user.PasswordHash = hash
	}

	var duplicate int64
	database.DB.Model(&models.User{}).
		Where("school_id = ? AND id <> ? AND (LOWER(username) = ? OR LOWER(email) = ?)", schoolID, user.ID, strings.ToLower(user.Username), strings.ToLower(user.Email)).
		Count(&duplicate)
	if duplicate > 0 {
		fail(c, http.StatusConflict, "username or email already exists")
		return
	}

	if err := database.DB.Save(&user).Error; err != nil {
		fail(c, http.StatusInternalServerError, "failed to update user")
		return
	}
	auditAction(c, "users", "update", "users", &user.ID)
	success(c, http.StatusOK, userPayload(user, nextRoleName), "User updated")
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == c.GetString("user_id") {
		fail(c, http.StatusBadRequest, "you cannot delete your own account")
		return
	}
	schoolID := scopedSchoolID(c)
	var user models.User
	if err := database.DB.Preload("Role").Where("id = ? AND school_id = ?", id, schoolID).First(&user).Error; err != nil {
		fail(c, http.StatusNotFound, "user not found")
		return
	}
	roleName := ""
	if user.Role != nil {
		roleName = user.Role.RoleName
	}
	if !canManageUserRole(c, roleName) {
		fail(c, http.StatusForbidden, "You cannot delete this user")
		return
	}
	if err := database.DB.Delete(&models.User{}, "id = ? AND school_id = ?", id, schoolID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "failed to delete user")
		return
	}
	auditAction(c, "users", "delete", "users", &id)
	success(c, http.StatusOK, nil, "User deleted")
}

func canManageUserRole(c *gin.Context, roleName string) bool {
	role := strings.ToLower(strings.TrimSpace(roleName))
	switch currentRole(c) {
	case "principal":
		return role == "admin" || role == "teacher"
	case "admin":
		return role == "parent"
	default:
		return false
	}
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "principal":
		return "Principal"
	case "admin":
		return "Admin"
	case "teacher":
		return "Teacher"
	case "parent":
		return "Parent"
	default:
		return ""
	}
}

func splitName(fullName, fallback string) (string, string) {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return fallback, "User"
	}
	if len(parts) == 1 {
		return parts[0], "User"
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func staffCode(role, username string) string {
	prefix := "STF"
	if role == "Admin" {
		prefix = "ADM"
	}
	clean := strings.NewReplacer(" ", "", "@", "", ".", "", "-", "").Replace(username)
	if len(clean) > 12 {
		clean = clean[:12]
	}
	return prefix + "-" + clean
}

func userPayload(u models.User, roleName string) gin.H {
	return gin.H{
		"id":          u.ID,
		"username":    u.Username,
		"email":       u.Email,
		"phone":       u.Phone,
		"school_id":   u.SchoolID,
		"role_id":     u.RoleID,
		"role_name":   roleName,
		"is_active":   u.IsActive,
		"is_verified": u.IsVerified,
		"last_login":  u.LastLogin,
		"created_at":  u.CreatedAt,
	}
}
