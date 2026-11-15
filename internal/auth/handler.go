package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/fickysp/project-hub-backend.git/config"
	"github.com/fickysp/project-hub-backend.git/internal/helper"
	"github.com/fickysp/project-hub-backend.git/pkg/sentinel"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authUsecase AuthUsecase
	cfg         *config.Config
}

func NewAuthHandler(router *gin.RouterGroup, au AuthUsecase, cfg *config.Config) {
	handler := &AuthHandler{
		authUsecase: au,
		cfg:         cfg,
	}

	group := router.Group("/auth")
	{
		group.POST("/register", handler.Register)
		group.POST("/login", handler.Login)
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if valErr := helper.FormatValidationError(err); valErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Validasi input tidak valid",
				"errors":  valErr,
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Format JSON tidak valid",
			"data":    nil,
		})
		return
	}

	if err := h.authUsecase.Register(c.Request.Context(), req); err != nil {
		if errors.Is(err, sentinel.ErrEmailIsUsed) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}

		slog.Error("Gagal melakukan registrasi user", "error", err,
			slog.String("email", req.Email))

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Tejadi kesalahan pada server",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Registrasi berhasil",
		"data":    nil,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if valErr := helper.FormatValidationError(err); valErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Validasi input tidak valid",
				"errors":  valErr,
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Format JSON tidak valid",
			"data":    nil,
		})
		return
	}

	resp, err := h.authUsecase.Login(
		c.Request.Context(),
		h.cfg.JWTSecret,
		h.cfg.RefreshSecret,
		req,
	)

	if err != nil {
		slog.Error("Failed to login", "error", err.Error(),
			slog.String("email", req.Email))
		if errors.Is(err, sentinel.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Email atau password salah",
				"data":    nil,
			})
			return

		} else if errors.Is(err, sentinel.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "User tidak ditemukan",
				"data":    nil,
			})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Terjadi kesalahan pada server",
				"data":    nil,
			})
			return
		}
	}

	maxAgeRefresh := 3600 * 24
	isSecure := false
	if h.cfg != nil && h.cfg.AppEnv == "production" {
		isSecure = true
	}

	c.SetCookie("refresh_token", resp.RefreshToken, maxAgeRefresh, "/", "", isSecure, true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login berhasil",
		"data":    resp,
	})
}
