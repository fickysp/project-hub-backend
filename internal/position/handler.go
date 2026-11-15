package projectposition

import (
	"log/slog"
	"net/http"

	"github.com/fickysp/project-hub-backend.git/internal/helper"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PositionHandler struct {
	usecase PositionUsecase
	router  *gin.RouterGroup
}

func NewPositionHandler(route *gin.RouterGroup, u PositionUsecase) {
	handler := PositionHandler{
		usecase: u,
		router:  route,
	}

	protected := route.Group("/:project_id/position")
	{
		protected.POST("", handler.CreatePosition)
		protected.GET("", handler.GetAllPosition)
		protected.GET("/:position_id", handler.GetPosition)
		protected.PATCH("/:position_id", handler.UpdatePosition)
		protected.DELETE("/:position_id", handler.DeletePosition)
	}
}

func (h *PositionHandler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

func (h *PositionHandler) CreatePosition(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		slog.Warn("CreatePosition: project_id tidak valid", "project_id", c.Param("project_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID tidak valid",
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("CreatePosition: user tidak terautentikasi", "project_id", projectID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	var req CreatePositionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if valErr := helper.FormatValidationError(err); valErr != nil {
			slog.Warn("CreatePosition: validasi input gagal", "user_id", userID, "project_id", projectID, "errors", valErr)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Validasi input tidak valid",
				"errors":  valErr,
				"data":    nil,
			})
			return
		}
		slog.Warn("CreatePosition: format JSON tidak valid", "user_id", userID, "project_id", projectID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	if err := h.usecase.CreatePosition(c.Request.Context(), projectID, userID, req); err != nil {
		if err.Error() == "forbidden: hanya lead project yang dapat melakukan aksi ini" {
			slog.Warn("CreatePosition: akses ditolak, bukan lead project", "user_id", userID, "project_id", projectID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		slog.Error("CreatePosition: gagal membuat posisi", "user_id", userID, "project_id", projectID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("CreatePosition: posisi berhasil dibuat", "user_id", userID, "project_id", projectID, "nama_posisi", req.Name)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Posisi berhasil dibuat",
		"data":    nil,
	})
}

func (h *PositionHandler) UpdatePosition(c *gin.Context) {
	posID, err := uuid.Parse(c.Param("position_id"))
	if err != nil {
		slog.Warn("UpdatePosition: position_id tidak valid", "position_id", c.Param("position_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Position ID tidak valid",
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("UpdatePosition: user tidak terautentikasi", "position_id", posID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	var req UpdatePositionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if valErr := helper.FormatValidationError(err); valErr != nil {
			slog.Warn("UpdatePosition: validasi input gagal", "user_id", userID, "position_id", posID, "errors", valErr)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Validasi input tidak valid",
				"errors":  valErr,
				"data":    nil,
			})
			return
		}
		slog.Warn("UpdatePosition: format JSON tidak valid", "user_id", userID, "position_id", posID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	if err := h.usecase.UpdatePosition(c.Request.Context(), posID, userID, req); err != nil {
		if err.Error() == "forbidden: hanya lead project yang dapat melakukan aksi ini" {
			slog.Warn("UpdatePosition: akses ditolak, bukan lead project", "user_id", userID, "position_id", posID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		slog.Error("UpdatePosition: gagal memperbarui posisi", "user_id", userID, "position_id", posID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("UpdatePosition: posisi berhasil diperbarui", "user_id", userID, "position_id", posID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Posisi berhasil diperbarui",
		"data":    nil,
	})
}

func (h *PositionHandler) DeletePosition(c *gin.Context) {
	posID, err := uuid.Parse(c.Param("position_id"))
	if err != nil {
		slog.Warn("DeletePosition: position_id tidak valid", "position_id", c.Param("position_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Position ID tidak valid",
			"errors":  nil,
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("DeletePosition: user tidak terautentikasi", "position_id", posID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"errors":  nil,
			"data":    nil,
		})
		return
	}

	if err := h.usecase.DeletePosition(c.Request.Context(), posID, userID); err != nil {
		if err.Error() == "forbidden: hanya lead project yang dapat melakukan aksi ini" {
			slog.Warn("DeletePosition: akses ditolak, bukan lead project", "user_id", userID, "position_id", posID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
				"errors":  nil,
				"data":    nil,
			})
			return
		}
		slog.Error("DeletePosition: gagal menghapus posisi", "user_id", userID, "position_id", posID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"errors":  nil,
			"data":    nil,
		})
		return
	}

	slog.Info("DeletePosition: posisi berhasil dihapus", "user_id", userID, "position_id", posID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Posisi berhasil dihapus",
		"errors":  nil,
		"data":    nil,
	})
}

func (h *PositionHandler) GetPosition(c *gin.Context) {
	posID, err := uuid.Parse(c.Param("position_id"))
	if err != nil {
		slog.Warn("GetPosition: position_id tidak valid", "position_id", c.Param("position_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Position ID tidak valid",
			"errors":  nil,
			"data":    nil,
		})
		return
	}

	result, err := h.usecase.GetPosition(c.Request.Context(), posID)
	if err != nil {
		if err.Error() == "posisi tidak ditemukan" {
			slog.Warn("GetPosition: posisi tidak ditemukan", "position_id", posID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": err.Error(),
				"errors":  nil,
				"data":    nil,
			})
			return
		}
		slog.Error("GetPosition: gagal mengambil data posisi", "position_id", posID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"errors":  nil,
			"data":    nil,
		})
		return
	}

	slog.Info("GetPosition: berhasil mengambil data posisi", "position_id", posID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Berhasil mendapatkan posisi",
		"errors":  nil,
		"data":    result,
	})
}

func (h *PositionHandler) GetAllPosition(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		slog.Warn("GetAllPosition: project_id tidak valid", "project_id", c.Param("project_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID tidak valid",
			"errors":  nil,
			"data":    nil,
		})
		return
	}

	result, err := h.usecase.GetAllPosition(c.Request.Context(), projectID)
	if err != nil {
		slog.Error("GetAllPosition: gagal mengambil semua posisi", "project_id", projectID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"errors":  nil,
			"data":    nil,
		})
		return
	}

	slog.Info("GetAllPosition: berhasil mengambil semua posisi", "project_id", projectID, "jumlah", len(result))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Berhasil mendapatkan semua posisi",
		"errors":  nil,
		"data":    result,
	})
}
