package selection

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
	group := router.Group("/projects/:project_id/applications")

	group.GET("", h.List)
	group.GET("/summary", h.Summary)
	group.GET("/:application_id", h.Detail)
	group.POST("/:application_id/review", h.Review)
	group.POST("/:application_id/accept", h.Accept)
	group.POST("/:application_id/reject", h.Reject)
}

func (h *Handler) List(c *gin.Context) {
	projectID, _, leadID, ok := getSelectionIDs(c, false)
	if !ok {
		return
	}

	var request ListApplicationsRequest

	if err := c.ShouldBindQuery(&request); err != nil {
		writeSelectionError(
			c,
			http.StatusBadRequest,
			"INVALID_QUERY",
			"Invalid search, position_id, status, page, or limit",
		)
		return
	}

	response, err := h.usecase.List(
		c.Request.Context(),
		projectID,
		leadID,
		request,
	)
	if err != nil {
		handleSelectionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Applications retrieved successfully",
		"data":       response.Data,
		"pagination": response.Pagination,
	})
}

func (h *Handler) Detail(c *gin.Context) {
	projectID, applicationID, leadID, ok := getSelectionIDs(c, true)
	if !ok {
		return
	}

	response, err := h.usecase.Detail(
		c.Request.Context(),
		projectID,
		applicationID,
		leadID,
	)
	if err != nil {
		handleSelectionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Application retrieved successfully",
		"data":    response,
	})
}

func (h *Handler) Summary(c *gin.Context) {
	projectID, _, leadID, ok := getSelectionIDs(c, false)
	if !ok {
		return
	}

	response, err := h.usecase.Summary(
		c.Request.Context(),
		projectID,
		leadID,
	)
	if err != nil {
		handleSelectionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Application summary retrieved successfully",
		"data":    response,
	})
}

func (h *Handler) Review(c *gin.Context) {
	projectID, applicationID, leadID, ok := getSelectionIDs(c, true)
	if !ok {
		return
	}

	response, err := h.usecase.Review(
		c.Request.Context(),
		projectID,
		applicationID,
		leadID,
	)
	if err != nil {
		handleSelectionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Application marked as reviewing",
		"data":    response,
	})
}

func (h *Handler) Accept(c *gin.Context) {
	projectID, applicationID, leadID, ok := getSelectionIDs(c, true)
	if !ok {
		return
	}

	var request AcceptApplicationRequest

	if !bindSelectionJSON(c, &request) {
		return
	}

	response, err := h.usecase.Accept(
		c.Request.Context(),
		projectID,
		applicationID,
		leadID,
		request,
	)
	if err != nil {
		handleSelectionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Application accepted successfully",
		"data":    response,
	})
}

func (h *Handler) Reject(c *gin.Context) {
	projectID, applicationID, leadID, ok := getSelectionIDs(c, true)
	if !ok {
		return
	}

	var request RejectApplicationRequest

	if !bindSelectionJSON(c, &request) {
		return
	}

	response, err := h.usecase.Reject(
		c.Request.Context(),
		projectID,
		applicationID,
		leadID,
		request,
	)
	if err != nil {
		handleSelectionError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Application rejected successfully",
		"data":    response,
	})
}

func getSelectionIDs(
	c *gin.Context,
	withApplication bool,
) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	leadID, ok := getSelectionUserID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil || projectID == uuid.Nil {
		writeSelectionError(
			c,
			http.StatusBadRequest,
			"INVALID_PROJECT_ID",
			"Project ID must be a valid non-empty UUID",
		)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	if !withApplication {
		return projectID, uuid.Nil, leadID, true
	}

	applicationID, err := uuid.Parse(c.Param("application_id"))
	if err != nil || applicationID == uuid.Nil {
		writeSelectionError(
			c,
			http.StatusBadRequest,
			"INVALID_APPLICATION_ID",
			"Application ID must be a valid non-empty UUID",
		)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	return projectID, applicationID, leadID, true
}

func getSelectionUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		handleSelectionError(c, sentinel.ErrUnauthorized)
		return uuid.Nil, false
	}

	var userID uuid.UUID

	switch value := value.(type) {
	case string:
		parsed, err := uuid.Parse(value)
		if err != nil {
			handleSelectionError(c, sentinel.ErrUnauthorized)
			return uuid.Nil, false
		}

		userID = parsed

	case uuid.UUID:
		userID = value

	default:
		handleSelectionError(c, sentinel.ErrUnauthorized)
		return uuid.Nil, false
	}

	if userID == uuid.Nil {
		handleSelectionError(c, sentinel.ErrUnauthorized)
		return uuid.Nil, false
	}

	return userID, true
}

func bindSelectionJSON(c *gin.Context, request interface{}) bool {
	if c.ContentType() != "application/json" {
		writeSelectionError(
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
			writeSelectionError(
				c,
				http.StatusRequestEntityTooLarge,
				"BODY_TOO_LARGE",
				"Request body must not exceed 1 MB",
			)
			return false
		}

		writeSelectionError(
			c,
			http.StatusBadRequest,
			"INVALID_BODY",
			"Unable to read request body",
		)
		return false
	}

	var fields map[string]json.RawMessage

	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		writeSelectionError(
			c,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Request body must be a valid JSON object",
		)
		return false
	}

	for field, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			handleSelectionError(c, &ValidationError{
				Field:   field,
				Message: "must not be null",
			})
			return false
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(request); err != nil {
		writeSelectionError(
			c,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Request contains an unknown field or invalid value type",
		)
		return false
	}

	if err := binding.Validator.ValidateStruct(request); err != nil {
		writeSelectionError(
			c,
			http.StatusUnprocessableEntity,
			"VALIDATION_ERROR",
			err.Error(),
		)
		return false
	}

	return true
}

func handleSelectionError(c *gin.Context, err error) {
	var validationError *ValidationError

	if errors.As(err, &validationError) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": validationError.Message,
			"error": gin.H{
				"code":  "VALIDATION_ERROR",
				"field": validationError.Field,
			},
		})
		return
	}

	status := http.StatusInternalServerError
	code := "INTERNAL_SERVER_ERROR"
	message := sentinel.ErrInternalServerError.Error()

	switch {
	case errors.Is(err, sentinel.ErrUnauthorized):
		status = http.StatusUnauthorized
		code = "UNAUTHORIZED"
		message = "Authentication required"

	case errors.Is(err, sentinel.ErrForbidden):
		status = http.StatusForbidden
		code = "FORBIDDEN"
		message = sentinel.ErrForbidden.Error()

	case errors.Is(err, sentinel.ErrProjectNotFound):
		status = http.StatusNotFound
		code = "PROJECT_NOT_FOUND"
		message = sentinel.ErrProjectNotFound.Error()

	case errors.Is(err, sentinel.ErrApplicationNotFound):
		status = http.StatusNotFound
		code = "APPLICATION_NOT_FOUND"
		message = sentinel.ErrApplicationNotFound.Error()

	case errors.Is(err, sentinel.ErrPositionNotFound):
		status = http.StatusNotFound
		code = "POSITION_NOT_FOUND"
		message = sentinel.ErrPositionNotFound.Error()

	case errors.Is(err, sentinel.ErrInvalidApplicationStatus):
		status = http.StatusConflict
		code = "INVALID_APPLICATION_STATUS"
		message = sentinel.ErrInvalidApplicationStatus.Error()

	case errors.Is(err, sentinel.ErrPositionFull):
		status = http.StatusConflict
		code = "POSITION_FULL"
		message = sentinel.ErrPositionFull.Error()

	case errors.Is(err, sentinel.ErrAlreadyProjectMember):
		status = http.StatusConflict
		code = "ALREADY_PROJECT_MEMBER"
		message = sentinel.ErrAlreadyProjectMember.Error()

	case errors.Is(err, sentinel.ErrProjectCompleted):
		status = http.StatusConflict
		code = "PROJECT_COMPLETED"
		message = sentinel.ErrProjectCompleted.Error()

	default:
		slog.Error(
			"Selection operation failed",
			"method", c.Request.Method,
			"route", c.FullPath(),
			"error", err,
		)
	}

	writeSelectionError(c, status, code, message)
}

func writeSelectionError(
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