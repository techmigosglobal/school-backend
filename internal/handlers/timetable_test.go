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

func TestCreateSubstitution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.NoError(t, database.SetupTestDB())

	reqBody := map[string]interface{}{
		"timetable_slot_id":   "test-slot-id",
		"date":                "2024-04-30",
		"original_staff_id":   "original-staff-id",
		"substitute_staff_id": "sub-staff-id",
		"reason":              "Sick leave",
		"approved_by":         "principal-id",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/timetable/substitutions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := gin.Default()
	r.POST("/timetable/substitutions", func(c *gin.Context) {
		h := NewTimetableHandler()
		h.CreateSubstitution(c)
	})

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response.Success)

	sub := response.Data.(map[string]interface{})
	assert.Equal(t, "test-slot-id", sub["timetable_slot_id"])
	assert.Equal(t, "2024-04-30T00:00:00Z", sub["date"])
}
