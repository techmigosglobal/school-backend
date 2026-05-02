package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

func success(c *gin.Context, status int, data interface{}, message string) {
	resp := models.APIResponse{
		Success:   true,
		Data:      data,
		RequestID: c.GetString("request_id"),
	}
	if message != "" {
		resp.Message = message
	}
	c.JSON(status, resp)
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, models.APIResponse{
		Success:   false,
		Code:      fmt.Sprintf("HTTP_%d", status),
		Message:   message,
		Error:     message,
		RequestID: c.GetString("request_id"),
	})
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func parseDateTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}

func paginationResult(page, pageSize int, total int64, data interface{}) models.PaginatedResponse {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return models.PaginatedResponse{
		Success:    true,
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func auditAction(c *gin.Context, module, action, tableName string, recordID *string) {
	uid := c.GetString("user_id")
	if uid == "" {
		return
	}
	log := models.AuditLog{
		UserID:     uid,
		Role:       c.GetString("role_name"),
		Module:     module,
		Action:     action,
		EntityType: tableName,
		EntityID:   recordID,
		TableName:  tableName,
		RecordID:   recordID,
		IPAddress:  c.ClientIP(),
		CreatedAt:  time.Now(),
	}
	_ = database.DB.Create(&log).Error
}

func parsePagination(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if v := c.Query("page_size"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 1 && parsed <= 100 {
			pageSize = parsed
		}
	}
	return page, pageSize
}

// scopedSchoolID returns the school_id extracted exclusively from the JWT
// claim set by AuthMiddleware. Query-param fallback has been intentionally
// removed: accepting a client-supplied school_id would allow cross-tenant
// data access if the token carries no school scope.
//
// The 403 gate for a missing claim is owned by SchoolScopeMiddleware, which
// aborts the request before any handler runs on protected route groups.
func scopedSchoolID(c *gin.Context) string {
	return strings.TrimSpace(c.GetString("school_id"))
}

func monthYearRange(month, year string) (time.Time, time.Time, bool) {
	cleanMonth := strings.TrimSpace(month)
	cleanYear := strings.TrimSpace(year)
	if cleanMonth == "" {
		return time.Time{}, time.Time{}, false
	}
	if cleanYear == "" {
		cleanYear = time.Now().UTC().Format("2006")
	}
	start, err := time.Parse("2006-01", cleanYear+"-"+cleanMonth)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	end := start.AddDate(0, 1, 0)
	return start, end, true
}
