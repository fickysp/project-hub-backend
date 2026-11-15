package member

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	usecase Usecase
}

func NewHandler(usecase Usecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST(
		"/projects/:project_id/leave",
		h.Leave,
	)

	router.DELETE(
		"/projects/:project_id/members/:member_id",
		h.Remove,
	)
}

func (h *Handler) Leave(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		handleError(c, ErrUnauthorized)
		return
	}

	projectID, ok := parseID(c, "project_id")
	if !ok {
		return
	}

	response, err := h.usecase.Leave(
		c.Request.Context(),
		projectID,
		userID,
	)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "You have left the project",
		"data":    response,
	})
}

func (h *Handler) Remove(c *gin.Context) {
	actorID, ok := authenticatedUserID(c)
	if !ok {
		handleError(c, ErrUnauthorized)
		return
	}

	projectID, ok := parseID(c, "project_id")
	if !ok {
		return
	}

	memberID, ok := parseID(c, "member_id")
	if !ok {
		return
	}

	response, err := h.usecase.Remove(
		c.Request.Context(),
		projectID,
		memberID,
		actorID,
	)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project member removed successfully",
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

func parseID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil || id == uuid.Nil {
		writeError(
			c,
			http.StatusBadRequest,
			"INVALID_ID",
			name+" must be a valid non-empty UUID",
		)
		return uuid.Nil, false
	}

	return id, true
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

	case errors.Is(err, ErrUseLeaveEndpoint):
		status = http.StatusConflict
		code = "USE_LEAVE_ENDPOINT"
		message = ErrUseLeaveEndpoint.Error()

	case errors.Is(err, ErrInvalidID):
		status = http.StatusBadRequest
		code = "INVALID_ID"
		message = ErrInvalidID.Error()

	default:
		slog.Error(
			"Failed to end project membership",
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