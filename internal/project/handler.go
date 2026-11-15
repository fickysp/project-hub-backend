package project

import (
	"bytes"
	"encoding/json"
	"io"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fickysp/project-hub-backend.git/pkg/sentinel"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
)

type Handler struct {
	usecase Usecase
}

func NewHandler(usecase Usecase) *Handler {
	return &Handler{
		usecase: usecase,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/projects", h.Create)
	router.PATCH("/projects/:project_id", h.Update)
	router.DELETE("/projects/:project_id", h.Delete)
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var request CreateProjectRequest

	if !bindProjectJSON(c, &request) {
		return
	}

	response, err := h.usecase.Create(
		c.Request.Context(),
		userID,
		request,
	)
	if err != nil {
		handleProjectError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Project created successfully",
		"data":    response,
	})
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		writeProjectError(
			c,
			http.StatusBadRequest,
			"INVALID_PROJECT_ID",
			"Project ID must be a valid UUID",
		)
		return
	}

	var request UpdateProjectRequest

	if !bindProjectJSON(c, &request) {
		return
	}

	response, err := h.usecase.Update(
		c.Request.Context(),
		projectID,
		userID,
		request,
	)
	if err != nil {
		handleProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Project updated successfully",
		"data":    response,
	})
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		writeProjectError(
			c,
			http.StatusBadRequest,
			"INVALID_PROJECT_ID",
			"Project ID must be a valid UUID",
		)
		return
	}

	if err := h.usecase.Delete(
		c.Request.Context(),
		projectID,
		userID,
	); err != nil {
		handleProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Project deleted successfully",
	})
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		return
	}

	var request ListProjectsRequest

	if err := c.ShouldBindQuery(&request); err != nil {
		writeProjectError(
			c,
			http.StatusBadRequest,
			"INVALID_QUERY",
			"Invalid search, status, skill_id, page, or limit",
		)
		return
	}

	response, err := h.usecase.List(
		c.Request.Context(),
		userID,
		request,
	)
	if err != nil {
		handleProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Projects retrieved successfully",
		"data":       response.Data,
		"pagination": response.Pagination,
	})
}

func getAuthenticatedUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		handleProjectError(c, sentinel.ErrUnauthorized)
		return uuid.Nil, false
	}

	var userID uuid.UUID

	switch value := value.(type) {
	case string:
		parsed, err := uuid.Parse(value)
		if err != nil {
			handleProjectError(c, sentinel.ErrUnauthorized)
			return uuid.Nil, false
		}

		userID = parsed

	case uuid.UUID:
		userID = value

	default:
		handleProjectError(c, sentinel.ErrUnauthorized)
		return uuid.Nil, false
	}

	if userID == uuid.Nil {
		handleProjectError(c, sentinel.ErrUnauthorized)
		return uuid.Nil, false
	}

	return userID, true
}

func bindProjectJSON(c *gin.Context, request interface{}) bool {
	if c.ContentType() != "application/json" {
		writeProjectError(
			c,
			http.StatusUnsupportedMediaType,
			"UNSUPPORTED_MEDIA_TYPE",
			"Content-Type must be application/json",
		)
		return false
	}

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		1<<20,
	)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var sizeError *http.MaxBytesError

		if errors.As(err, &sizeError) {
			writeProjectError(
				c,
				http.StatusRequestEntityTooLarge,
				"BODY_TOO_LARGE",
				"Request body must not exceed 1 MB",
			)
			return false
		}

		writeProjectError(
			c,
			http.StatusBadRequest,
			"INVALID_BODY",
			"Unable to read request body",
		)
		return false
	}

	var fields map[string]json.RawMessage

	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		writeProjectError(
			c,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Request body must be a valid JSON object",
		)
		return false
	}

	for field, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			handleProjectError(c, &ValidationError{
				Field:   field,
				Message: "must not be null; omit unchanged fields or use [] to clear skills",
			})
			return false
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(request); err != nil {
		writeProjectError(
			c,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Request contains an unknown field or invalid value type",
		)
		return false
	}

	if err := binding.Validator.ValidateStruct(request); err != nil {
		writeProjectError(
			c,
			http.StatusUnprocessableEntity,
			"VALIDATION_ERROR",
			err.Error(),
		)
		return false
	}

	return true
}

func handleProjectError(c *gin.Context, err error) {
	var validationError *ValidationError

	switch {
	case errors.As(err, &validationError):
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": validationError.Message,
			"error": gin.H{
				"code":  "VALIDATION_ERROR",
				"field": validationError.Field,
			},
		})

	case errors.Is(err, sentinel.ErrUnauthorized):
		writeProjectError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication required",
		)

	case errors.Is(err, sentinel.ErrProjectNotFound):
		writeProjectError(
			c,
			http.StatusNotFound,
			"PROJECT_NOT_FOUND",
			"Project not found",
		)

	case errors.Is(err, sentinel.ErrForbidden):
		writeProjectError(
			c,
			http.StatusForbidden,
			"FORBIDDEN",
			"Only an active project lead can perform this action",
		)

	case errors.Is(err, sentinel.ErrPendingApplications):
		writeProjectError(
			c,
			http.StatusConflict,
			"PENDING_APPLICATIONS",
			"Resolve pending applications before completing the project",
		)

	case errors.Is(err, sentinel.ErrInvalidSkills):
		writeProjectError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_SKILLS",
			"One or more skills do not exist",
		)

	default:
		slog.Error(
			"Project operation failed",
			"method", c.Request.Method,
			"route", c.FullPath(),
			"error", err,
		)

		writeProjectError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_SERVER_ERROR",
			"Internal server error",
		)
	}
}

func writeProjectError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(status, gin.H{
		"message": message,
		"error": gin.H{
			"code": code,
		},
	})
}