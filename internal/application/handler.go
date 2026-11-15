package application

import (
	"log/slog"
	"net/http"

	"github.com/fickysp/project-hub-backend.git/internal/helper"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ApplicationHandler struct {
	usecase ApplicationUsecase
}

func NewApplicationHandler(route *gin.RouterGroup, u ApplicationUsecase) {
	handler := ApplicationHandler{usecase: u}

	route.GET("/me/applications", handler.GetMyApplications)

	project := route.Group("/projects/:project_id")
	{
		project.POST("/applications", handler.SubmitApplication)
		project.GET("/applications", handler.GetApplicationsByProject)
		project.GET("/applications/:application_id", handler.GetApplication)
		project.PATCH("/applications/:application_id/review", handler.ReviewApplication)
		project.PATCH("/applications/:application_id/cancel", handler.CancelApplication)
		project.GET("/positions/:position_id/applications", handler.GetApplicationsByPosition)
	}
}

func (h *ApplicationHandler) getUserID(c *gin.Context) (uuid.UUID, bool) {
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

func (h *ApplicationHandler) SubmitApplication(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		slog.Warn("SubmitApplication: project_id tidak valid", "project_id", c.Param("project_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID tidak valid",
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("SubmitApplication: user tidak terautentikasi", "project_id", projectID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	var req SubmitApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if valErr := helper.FormatValidationError(err); valErr != nil {
			slog.Warn("SubmitApplication: validasi input gagal", "user_id", userID, "project_id", projectID, "errors", valErr)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Validasi input tidak valid",
				"errors":  valErr,
				"data":    nil,
			})
			return
		}
		slog.Warn("SubmitApplication: format JSON tidak valid", "user_id", userID, "project_id", projectID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	if err := h.usecase.SubmitApplication(c.Request.Context(), projectID, userID, req); err != nil {
		if err.Error() == "posisi sudah penuh atau tidak lagi menerima pengajuan" {
			slog.Warn("SubmitApplication: posisi penuh", "user_id", userID, "project_id", projectID)
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		if err.Error() == "kamu sudah memiliki pengajuan yang sedang diproses pada project ini" {
			slog.Warn("SubmitApplication: duplikat pengajuan", "user_id", userID, "project_id", projectID)
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		slog.Error("SubmitApplication: gagal submit pengajuan", "user_id", userID, "project_id", projectID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("SubmitApplication: pengajuan berhasil dikirim", "user_id", userID, "project_id", projectID)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Pengajuan berhasil dikirim",
		"data":    nil,
	})
}

func (h *ApplicationHandler) ReviewApplication(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		slog.Warn("ReviewApplication: application_id tidak valid", "application_id", c.Param("application_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Application ID tidak valid",
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("ReviewApplication: user tidak terautentikasi", "application_id", appID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	var req ReviewApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if valErr := helper.FormatValidationError(err); valErr != nil {
			slog.Warn("ReviewApplication: validasi input gagal", "user_id", userID, "application_id", appID, "errors", valErr)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Validasi input tidak valid",
				"errors":  valErr,
				"data":    nil,
			})
			return
		}
		slog.Warn("ReviewApplication: format JSON tidak valid", "user_id", userID, "application_id", appID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	if err := h.usecase.ReviewApplication(c.Request.Context(), appID, userID, req); err != nil {
		if err.Error() == "forbidden: hanya lead project yang dapat melakukan aksi ini" {
			slog.Warn("ReviewApplication: akses ditolak", "user_id", userID, "application_id", appID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		if err.Error() == "pengajuan tidak ditemukan" {
			slog.Warn("ReviewApplication: pengajuan tidak ditemukan", "application_id", appID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		if err.Error() == "pengajuan sudah diputuskan dan tidak dapat diubah lagi" {
			slog.Warn("ReviewApplication: pengajuan sudah diputuskan", "application_id", appID)
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		slog.Error("ReviewApplication: gagal mereview pengajuan", "user_id", userID, "application_id", appID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("ReviewApplication: pengajuan berhasil direview", "user_id", userID, "application_id", appID, "status", req.Status)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Pengajuan berhasil direview",
		"data":    nil,
	})
}

func (h *ApplicationHandler) CancelApplication(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		slog.Warn("CancelApplication: application_id tidak valid", "application_id", c.Param("application_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Application ID tidak valid",
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("CancelApplication: user tidak terautentikasi", "application_id", appID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	if err := h.usecase.CancelApplication(c.Request.Context(), appID, userID); err != nil {
		if err.Error() == "forbidden: kamu tidak memiliki akses untuk membatalkan pengajuan ini" {
			slog.Warn("CancelApplication: akses ditolak", "user_id", userID, "application_id", appID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		if err.Error() == "pengajuan tidak ditemukan" {
			slog.Warn("CancelApplication: pengajuan tidak ditemukan", "application_id", appID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		if err.Error() == "pengajuan sudah diputuskan dan tidak dapat dibatalkan" {
			slog.Warn("CancelApplication: pengajuan sudah diputuskan", "application_id", appID)
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		slog.Error("CancelApplication: gagal membatalkan pengajuan", "user_id", userID, "application_id", appID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("CancelApplication: pengajuan berhasil dibatalkan", "user_id", userID, "application_id", appID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Pengajuan berhasil dibatalkan",
		"data":    nil,
	})
}

func (h *ApplicationHandler) GetApplication(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		slog.Warn("GetApplication: application_id tidak valid", "application_id", c.Param("application_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Application ID tidak valid",
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("GetApplication: user tidak terautentikasi", "application_id", appID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	result, err := h.usecase.GetApplication(c.Request.Context(), appID, userID)
	if err != nil {
		if err.Error() == "pengajuan tidak ditemukan" {
			slog.Warn("GetApplication: pengajuan tidak ditemukan", "application_id", appID)
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		if err.Error() == "forbidden: kamu tidak memiliki akses untuk melihat pengajuan ini" {
			slog.Warn("GetApplication: akses ditolak", "user_id", userID, "application_id", appID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		slog.Error("GetApplication: gagal mengambil data pengajuan", "application_id", appID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("GetApplication: berhasil mengambil data pengajuan", "application_id", appID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Berhasil mendapatkan pengajuan",
		"data":    result,
	})
}

func (h *ApplicationHandler) GetApplicationsByProject(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		slog.Warn("GetApplicationsByProject: project_id tidak valid", "project_id", c.Param("project_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID tidak valid",
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("GetApplicationsByProject: user tidak terautentikasi", "project_id", projectID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	result, err := h.usecase.GetApplicationsByProject(c.Request.Context(), projectID, userID)
	if err != nil {
		if err.Error() == "forbidden: hanya lead project yang dapat melihat semua pengajuan" {
			slog.Warn("GetApplicationsByProject: akses ditolak", "user_id", userID, "project_id", projectID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		slog.Error("GetApplicationsByProject: gagal mengambil data", "project_id", projectID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("GetApplicationsByProject: berhasil mengambil semua pengajuan", "project_id", projectID, "jumlah", len(result))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Berhasil mendapatkan semua pengajuan",
		"data":    result,
	})
}

func (h *ApplicationHandler) GetMyApplications(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("GetMyApplications: user tidak terautentikasi")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	result, err := h.usecase.GetApplicationsByApplicant(c.Request.Context(), userID)
	if err != nil {
		slog.Error("GetMyApplications: gagal mengambil data pengajuan", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("GetMyApplications: berhasil mengambil pengajuan milik user", "user_id", userID, "jumlah", len(result))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Berhasil mendapatkan pengajuan kamu",
		"data":    result,
	})
}

func (h *ApplicationHandler) GetApplicationsByPosition(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		slog.Warn("GetApplicationsByPosition: project_id tidak valid", "project_id", c.Param("project_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID tidak valid",
			"data":    nil,
		})
		return
	}

	positionID, err := uuid.Parse(c.Param("position_id"))
	if err != nil {
		slog.Warn("GetApplicationsByPosition: position_id tidak valid", "position_id", c.Param("position_id"))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Position ID tidak valid",
			"data":    nil,
		})
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		slog.Warn("GetApplicationsByPosition: user tidak terautentikasi", "position_id", positionID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
			"data":    nil,
		})
		return
	}

	result, err := h.usecase.GetApplicationsByPosition(c.Request.Context(), projectID, positionID, userID)
	if err != nil {
		if err.Error() == "forbidden: hanya lead project yang dapat melihat pengajuan per posisi" {
			slog.Warn("GetApplicationsByPosition: akses ditolak", "user_id", userID, "position_id", positionID)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		slog.Error("GetApplicationsByPosition: gagal mengambil data", "position_id", positionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	slog.Info("GetApplicationsByPosition: berhasil mengambil pengajuan per posisi", "position_id", positionID, "jumlah", len(result))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Berhasil mendapatkan pengajuan per posisi",
		"data":    result,
	})
}
