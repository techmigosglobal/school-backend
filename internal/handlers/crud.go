package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"school-backend/internal/database"
	"school-backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CRUDHandler[T any] struct {
	Module       string
	TableName    string
	Required     []string
	SchoolScoped bool
	Preloads     []string
}

func NewCRUDHandler[T any](module, tableName string, required []string, schoolScoped bool, preloads ...string) *CRUDHandler[T] {
	return &CRUDHandler[T]{
		Module:       module,
		TableName:    tableName,
		Required:     required,
		SchoolScoped: schoolScoped,
		Preloads:     preloads,
	}
}

func (h *CRUDHandler[T]) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	offset := (page - 1) * pageSize

	var rows []T
	var total int64
	query := h.scopedQuery(c).Model(new(T))
	query = h.applyRoleScope(c, query)
	for _, preload := range h.Preloads {
		query = query.Preload(preload)
	}
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to count "+h.Module)
		return
	}
	if err := query.Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to list "+h.Module)
		return
	}
	c.JSON(http.StatusOK, paginationResult(page, pageSize, total, rows))
}

func (h *CRUDHandler[T]) Get(c *gin.Context) {
	var row T
	query := h.scopedQuery(c)
	query = h.applyRoleScope(c, query)
	for _, preload := range h.Preloads {
		query = query.Preload(preload)
	}
	if err := query.First(&row, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if currentRole(c) == "parent" || currentRole(c) == "teacher" {
				fail(c, http.StatusForbidden, h.Module+" access denied")
				return
			}
			fail(c, http.StatusNotFound, h.Module+" not found")
			return
		}
		fail(c, http.StatusInternalServerError, "Failed to load "+h.Module)
		return
	}
	success(c, http.StatusOK, row, "")
}

func (h *CRUDHandler[T]) Create(c *gin.Context) {
	var row T
	if err := h.bindAndValidate(c, &row, true); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.applySchoolScope(c, &row)
	if status, msg := h.validateOwnership(c, &row); status != http.StatusOK {
		fail(c, status, msg)
		return
	}
	if err := database.DB.Create(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to create "+h.Module)
		return
	}
	id := modelID(&row)
	auditAction(c, h.Module, "create", h.TableName, &id)
	success(c, http.StatusCreated, row, "")
}

func (h *CRUDHandler[T]) Update(c *gin.Context) {
	id := c.Param("id")
	var row T
	if err := h.applyRoleScope(c, h.scopedQuery(c)).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, h.Module+" not found")
			return
		}
		fail(c, http.StatusInternalServerError, "Failed to load "+h.Module)
		return
	}
	if err := h.bindAndValidate(c, &row, false); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	setStringField(&row, "ID", id)
	h.applySchoolScope(c, &row)
	if status, msg := h.validateOwnership(c, &row); status != http.StatusOK {
		fail(c, status, msg)
		return
	}
	if err := database.DB.Save(&row).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to update "+h.Module)
		return
	}
	auditAction(c, h.Module, "update", h.TableName, &id)
	success(c, http.StatusOK, row, "")
}

func (h *CRUDHandler[T]) Delete(c *gin.Context) {
	id := c.Param("id")
	result := h.applyRoleScope(c, h.scopedQuery(c)).Delete(new(T), "id = ?", id)
	if result.Error != nil {
		fail(c, http.StatusInternalServerError, "Failed to delete "+h.Module)
		return
	}
	if result.RowsAffected == 0 {
		fail(c, http.StatusNotFound, h.Module+" not found")
		return
	}
	auditAction(c, h.Module, "delete", h.TableName, &id)
	success(c, http.StatusOK, nil, h.Module+" deleted successfully")
}

func (h *CRUDHandler[T]) scopedQuery(c *gin.Context) *gorm.DB {
	query := database.DB
	if h.SchoolScoped {
		query = query.Where("school_id = ?", scopedSchoolID(c))
	}
	return query
}

func (h *CRUDHandler[T]) applyRoleScope(c *gin.Context, query *gorm.DB) *gorm.DB {
	role := currentRole(c)
	table := h.queryTableName()
	switch h.TableName {
	case "homework":
		switch role {
		case "parent":
			return query.Where(table+".student_id IN (?)", linkedStudentSubquery(c))
		case "teacher":
			return query.Where("("+table+".teacher_id = ? OR "+table+".section_id IN (?))", currentStaffID(c), teacherAssignedSectionSubquery(c))
		}
	case "diary_entries":
		switch role {
		case "parent":
			return query.Where("diary_entries.student_id IN (?)", linkedStudentSubquery(c))
		case "teacher":
			return query.Where("(diary_entries.teacher_id = ? OR diary_entries.section_id IN (?))", currentStaffID(c), teacherAssignedSectionSubquery(c))
		}
	case "message_conversations":
		switch role {
		case "parent":
			return query.Where("message_conversations.parent_id = ? AND message_conversations.student_id IN (?)", c.GetString("user_id"), linkedStudentSubquery(c))
		case "teacher":
			return query.Where("message_conversations.teacher_id = ? AND message_conversations.student_id IN (?)", currentStaffID(c), database.DB.Model(&models.Student{}).Select("students.id").Where("students.current_section_id IN (?)", teacherAssignedSectionSubquery(c)))
		}
	case "messages":
		query = query.Joins("JOIN message_conversations ON message_conversations.id = messages.conversation_id").
			Where("message_conversations.school_id = ?", scopedSchoolID(c))
		switch role {
		case "parent":
			return query.Where("message_conversations.parent_id = ? AND message_conversations.student_id IN (?)", c.GetString("user_id"), linkedStudentSubquery(c))
		case "teacher":
			return query.Where("message_conversations.teacher_id = ? AND message_conversations.student_id IN (?)", currentStaffID(c), database.DB.Model(&models.Student{}).Select("students.id").Where("students.current_section_id IN (?)", teacherAssignedSectionSubquery(c)))
		default:
			return query
		}
	case "guardians", "medical_records", "student_documents", "parent_teacher_meetings":
		if role == "parent" {
			return query.Where(h.TableName+".student_id IN (?)", linkedStudentSubquery(c))
		}
		if role == "teacher" {
			return query.Where(h.TableName+".student_id IN (?)", database.DB.Model(&models.Student{}).Select("students.id").Where("students.current_section_id IN (?)", teacherAssignedSectionSubquery(c)))
		}
	}
	return query
}

func (h *CRUDHandler[T]) queryTableName() string {
	if h.TableName == "homework" {
		return "homeworks"
	}
	return h.TableName
}

func (h *CRUDHandler[T]) validateOwnership(c *gin.Context, row *T) (int, string) {
	switch h.TableName {
	case "homework", "diary_entries":
		studentID := getStringField(row, "StudentID")
		sectionID := getStringField(row, "SectionID")
		teacherID := getStringField(row, "TeacherID")
		if studentID != "" && !canAccessStudentID(c, studentID) {
			return http.StatusForbidden, "student access denied"
		}
		if currentRole(c) == "teacher" {
			staffID := currentStaffID(c)
			if teacherID != "" && teacherID != staffID {
				return http.StatusForbidden, "teacher ownership denied"
			}
			if sectionID != "" && !teacherCanAccessSection(c, sectionID) {
				return http.StatusForbidden, "section access denied"
			}
		}
	case "message_conversations":
		studentID := getStringField(row, "StudentID")
		teacherID := getStringField(row, "TeacherID")
		parentID := getStringField(row, "ParentID")
		if studentID != "" && !canAccessStudentID(c, studentID) {
			return http.StatusForbidden, "student access denied"
		}
		if currentRole(c) == "parent" && parentID != c.GetString("user_id") {
			return http.StatusForbidden, "parent ownership denied"
		}
		if currentRole(c) == "teacher" && teacherID != currentStaffID(c) {
			return http.StatusForbidden, "teacher ownership denied"
		}
	case "messages":
		conversationID := getStringField(row, "ConversationID")
		senderID := getStringField(row, "SenderID")
		senderRole := strings.ToLower(getStringField(row, "SenderRole"))
		var convo models.MessageConversation
		convoQuery := database.DB.Model(&models.MessageConversation{}).Where("school_id = ?", scopedSchoolID(c))
		switch currentRole(c) {
		case "parent":
			convoQuery = convoQuery.Where("parent_id = ? AND student_id IN (?)", c.GetString("user_id"), linkedStudentSubquery(c))
		case "teacher":
			convoQuery = convoQuery.Where("teacher_id = ? AND student_id IN (?)", currentStaffID(c), database.DB.Model(&models.Student{}).Select("students.id").Where("students.current_section_id IN (?)", teacherAssignedSectionSubquery(c)))
		}
		if err := convoQuery.First(&convo, "id = ?", conversationID).Error; err != nil {
			return http.StatusForbidden, "conversation access denied"
		}
		if currentRole(c) == "parent" && (senderID != c.GetString("user_id") || senderRole != "parent") {
			return http.StatusForbidden, "sender ownership denied"
		}
		if currentRole(c) == "teacher" && (senderID != currentStaffID(c) || senderRole != "teacher") {
			return http.StatusForbidden, "sender ownership denied"
		}
	case "guardians", "medical_records", "student_documents", "parent_teacher_meetings":
		studentID := getStringField(row, "StudentID")
		if studentID != "" && !canAccessStudentID(c, studentID) {
			return http.StatusForbidden, "student access denied"
		}
	}
	return http.StatusOK, ""
}

func (h *CRUDHandler[T]) bindAndValidate(c *gin.Context, row *T, requireAll bool) error {
	raw, err := c.GetRawData()
	if err != nil {
		return errors.New("Invalid request body")
	}
	if len(raw) == 0 {
		return errors.New("Request body is required")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return errors.New("Invalid JSON body")
	}
	if requireAll {
		for _, field := range h.Required {
			if isEmptyJSONValue(payload[field]) {
				return fmt.Errorf("%s is required", field)
			}
		}
	}
	if err := json.Unmarshal(raw, row); err != nil {
		return errors.New("Invalid request fields")
	}
	return nil
}

func (h *CRUDHandler[T]) applySchoolScope(c *gin.Context, row *T) {
	if h.SchoolScoped {
		setStringField(row, "SchoolID", scopedSchoolID(c))
	}
}

func isEmptyJSONValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}

func setStringField(row interface{}, name, value string) {
	v := reflect.ValueOf(row)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}
	field := elem.FieldByName(name)
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
		field.SetString(value)
	}
}

func getStringField(row interface{}, name string) string {
	v := reflect.ValueOf(row)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	field := v.FieldByName(name)
	if field.IsValid() && field.Kind() == reflect.String {
		return strings.TrimSpace(field.String())
	}
	return ""
}

func modelID(row interface{}) string {
	v := reflect.ValueOf(row)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	field := v.FieldByName("ID")
	if field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	return ""
}

func NewAuditLogHandler() *AuditLogHandler {
	return &AuditLogHandler{}
}

type AuditLogHandler struct{}

func (h *AuditLogHandler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)
	offset := (page - 1) * pageSize
	var rows []models.AuditLog
	var total int64
	query := database.DB.Preload("User").Where(
		"user_id IN (?)",
		database.DB.Model(&models.User{}).Select("id").Where("school_id = ?", scopedSchoolID(c)),
	).Order("created_at DESC")
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}
	if err := query.Model(&models.AuditLog{}).Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to count audit logs")
		return
	}
	if err := query.Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to list audit logs")
		return
	}
	c.JSON(http.StatusOK, paginationResult(page, pageSize, total, rows))
}
