package member

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
	router.GET(
		"/projects/:project_id/members",
		h.List,
	)

	router.PATCH(
		"/projects/:project_id/members/:member_id/position",
		h.UpdatePosition,
	)

	router.POST(
		"/projects/:project_id/leave",
		h.Leave,
	)

	router.DELETE(
		"/projects/:project_id/members/:member_id",
		h.Remove,
	)
}

func (h *Handler) List(c *gin.Context) {
	projectID, userID, ok := requestAccess(c)
	if !ok {
		return
	}

	var request ListMembersRequest

	if err := c.ShouldBindQuery(&request); err != nil {
		handleError(c, ErrInvalidPagination)
		return
	}

	response, err := h.usecase.List(
		c.Request.Context(),
		projectID,
		userID,
		request,
	)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Project members retrieved successfully",
		"data":       response.Data,
		"pagination": response.Pagination,
	})
}

func (h *Handler) UpdatePosition(c *gin.Context) {
	projectID, actorID, ok := requestAccess(c)
	if !ok {
		return
	}

	memberID, ok := parseID(c, "member_id")
	if !ok {
		return
	}

	var request UpdateMemberPositionRequest

	if !bindPositionJSON(c, &request) {
		return
	}

	response, err := h.usecase.UpdatePosition(
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
		"message": "Member position updated successfully",
		"data":    response,
	})
}

func (h *Handler) Leave(c *gin.Context) {
	projectID, userID, ok := requestAccess(c)
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
	projectID, actorID, ok := requestAccess(c)
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

func requestAccess(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		handleError(c, ErrUnauthorized)
		return uuid.Nil, uuid.Nil, false
	}

	projectID, ok := parseID(c, "project_id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	return projectID, userID, true
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

func bindPositionJSON(
	c *gin.Context,
	request *UpdateMemberPositionRequest,
) bool {
	if c.ContentType() != "application/json" {
		writeError(
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
		16<<10,
	)

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(request); err != nil {
		handleBodyError(c, err)
		return false
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
		return false
	}

	if err := binding.Validator.ValidateStruct(request); err != nil {
		writeError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_POSITION_ID",
			"position_id must be a valid non-empty UUID",
		)
		return false
	}

	if request.PositionID == uuid.Nil {
		writeError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_POSITION_ID",
			"position_id must be a valid non-empty UUID",
		)
		return false
	}

	return true
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
		"Request body must be valid JSON containing only position_id",
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

	case errors.Is(err, ErrPositionNotFound):
		status = http.StatusNotFound
		code = "POSITION_NOT_FOUND"
		message = ErrPositionNotFound.Error()

	case errors.Is(err, ErrPositionFull):
		status = http.StatusConflict
		code = "POSITION_FULL"
		message = ErrPositionFull.Error()

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

	case errors.Is(err, ErrInvalidPagination):
		status = http.StatusBadRequest
		code = "INVALID_PAGINATION"
		message = ErrInvalidPagination.Error()

	default:
		slog.Error(
			"Project membership request failed",
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
