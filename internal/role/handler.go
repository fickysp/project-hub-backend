package role

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
)

type Handler struct {
	usecase Usecase
}

func NewHandler(usecase Usecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.PATCH(
		"/projects/:project_id/members/:member_id/role",
		h.UpdateMemberRole,
	)
}

func (h *Handler) UpdateMemberRole(c *gin.Context) {
	actorID, ok := authenticatedUserID(c)
	if !ok {
		writeError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			ErrUnauthorized.Error(),
		)
		return
	}

	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil || projectID == uuid.Nil {
		writeError(
			c,
			http.StatusBadRequest,
			"INVALID_PROJECT_ID",
			"Project ID must be a valid non-empty UUID",
		)
		return
	}

	memberID, err := uuid.Parse(c.Param("member_id"))
	if err != nil || memberID == uuid.Nil {
		writeError(
			c,
			http.StatusBadRequest,
			"INVALID_MEMBER_ID",
			"Member ID must be a valid non-empty UUID",
		)
		return
	}

	if c.ContentType() != "application/json" {
		writeError(
			c,
			http.StatusUnsupportedMediaType,
			"UNSUPPORTED_MEDIA_TYPE",
			"Content-Type must be application/json",
		)
		return
	}

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		16<<10,
	)

	var request UpdateMemberRoleRequest

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		handleBodyError(c, err)
		return
	}

	var extra interface{}

	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			handleBodyError(c, err)
		} else {
			writeError(
				c,
				http.StatusBadRequest,
				"INVALID_JSON",
				"Request body must contain one JSON object",
			)
		}
		return
	}

	if err := binding.Validator.ValidateStruct(request); err != nil {
		writeError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_ROLE",
			ErrInvalidRole.Error(),
		)
		return
	}

	response, err := h.usecase.UpdateMemberRole(
		c.Request.Context(),
		projectID,
		memberID,
		actorID,
		request,
	)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project member role updated successfully",
		"data":    response,
	})
}

func authenticatedUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}

	switch value := value.(type) {
	case string:
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			return uuid.Nil, false
		}
		return id, true

	case uuid.UUID:
		return value, value != uuid.Nil

	default:
		return uuid.Nil, false
	}
}

func handleBodyError(c *gin.Context, err error) {
	var sizeError *http.MaxBytesError

	if errors.As(err, &sizeError) {
		writeError(
			c,
			http.StatusRequestEntityTooLarge,
			"BODY_TOO_LARGE",
			"Request body must not exceed 16 KB",
		)
		return
	}

	writeError(
		c,
		http.StatusBadRequest,
		"INVALID_JSON",
		"Request body must be valid JSON containing only the role field",
	)
}

func handleError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL_SERVER_ERROR"
	message := "An internal server error occurred"

	switch {
	case errors.Is(err, ErrUnauthorized):
		status = http.StatusUnauthorized
		code = "UNAUTHORIZED"
		message = ErrUnauthorized.Error()

	case errors.Is(err, ErrForbidden):
		status = http.StatusForbidden
		code = "FORBIDDEN"
		message = ErrForbidden.Error()

	case errors.Is(err, ErrProjectNotFound):
		status = http.StatusNotFound
		code = "PROJECT_NOT_FOUND"
		message = ErrProjectNotFound.Error()

	case errors.Is(err, ErrMemberNotFound):
		status = http.StatusNotFound
		code = "MEMBER_NOT_FOUND"
		message = ErrMemberNotFound.Error()

	case errors.Is(err, ErrMemberInactive):
		status = http.StatusConflict
		code = "MEMBER_INACTIVE"
		message = ErrMemberInactive.Error()

	case errors.Is(err, ErrLastProjectLead):
		status = http.StatusConflict
		code = "LAST_PROJECT_LEAD"
		message = ErrLastProjectLead.Error()

	case errors.Is(err, ErrInvalidID):
		status = http.StatusBadRequest
		code = "INVALID_ID"
		message = ErrInvalidID.Error()

	case errors.Is(err, ErrInvalidRole):
		status = http.StatusUnprocessableEntity
		code = "INVALID_ROLE"
		message = ErrInvalidRole.Error()

	default:
		slog.Error(
			"Failed to update project member role",
			"error", err,
			"project_id", c.Param("project_id"),
			"member_id", c.Param("member_id"),
		)
	}

	writeError(c, status, code, message)
}

func writeError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(status, gin.H{
		"success": false,
		"message": message,
		"error": gin.H{
			"code": code,
		},
		"data": nil,
	})
}
