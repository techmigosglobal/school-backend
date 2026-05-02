package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateAttendanceSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.NoError(t, database.SetupTestDB())

	reqBody := map[string]interface{}{
		"section_id":    "test-section-id",
		"subject_id":    "test-subject-id",
		"staff_id":      "test-staff-id",
		"date":          "2024-04-30",
		"period_number": 1,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/attendance/sessions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := gin.Default()
	r.POST("/attendance/sessions", func(c *gin.Context) {
		h := NewAttendanceHandler()
		h.CreateAttendanceSession(c)
	})

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response.Success)

	session := response.Data.(map[string]interface{})
	assert.Equal(t, "test-section-id", session["section_id"])
	assert.Equal(t, "2024-04-30T00:00:00Z", session["date"]) // Assuming UTC
}

func TestMarkStaffAttendance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.NoError(t, database.SetupTestDB())

	reqBody := map[string]interface{}{
		"staff_id":     "test-staff-id",
		"date":         "2024-04-30",
		"status":       "present",
		"check_in":     "09:00:00",
		"check_out":    "17:00:00",
		"biometric_id": "bio123",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/attendance/staff", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := gin.Default()
	r.POST("/attendance/staff", func(c *gin.Context) {
		h := NewAttendanceHandler()
		h.MarkStaffAttendance(c)
	})

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response.Success)

	attendance := response.Data.(map[string]interface{})
	assert.Equal(t, "test-staff-id", attendance["staff_id"])
	assert.Equal(t, "2024-04-30T00:00:00Z", attendance["date"])
	assert.Equal(t, "2024-04-30T09:00:00Z", attendance["check_in"])
	assert.Equal(t, "2024-04-30T17:00:00Z", attendance["check_out"])
}
