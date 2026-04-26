package handlers

import (
	"math"
	"strconv"
	"time"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
)

func success(c *gin.Context, status int, data interface{}, message string) {
	resp := models.APIResponse{Success: true, Data: data}
	if message != "" {
		resp.Message = message
	}
	c.JSON(status, resp)
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, models.APIResponse{Success: false, Error: message})
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
		UserID:    uid,
		Module:    module,
		Action:    action,
		TableName: tableName,
		RecordID:  recordID,
		IPAddress: c.ClientIP(),
		CreatedAt: time.Now(),
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
