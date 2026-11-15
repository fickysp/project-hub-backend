package bootstrap

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/fickysp/project-hub-backend.git/config"
	"github.com/fickysp/project-hub-backend.git/infrastructure/database"
	"github.com/fickysp/project-hub-backend.git/internal/auth"
	"github.com/fickysp/project-hub-backend.git/internal/middleware"
	"github.com/fickysp/project-hub-backend.git/internal/project"
	"github.com/fickysp/project-hub-backend.git/internal/role"
	"github.com/fickysp/project-hub-backend.git/internal/member"
	"github.com/fickysp/project-hub-backend.git/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

func Run() error {
	logger.InitLogger()

	cfg := config.Load()

	db := database.Connect(cfg.DatabaseURL)

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer sqlDB.Close()

	blacklistCache := cache.New(
		15*time.Minute,
		5*time.Minute,
	)

	authMiddleware := middleware.NewAuthMiddleware(
		cfg.JWTSecret,
		blacklistCache,
	)

	authRepository := auth.NewAuthRepository(db)
	authUsecase := auth.NewAuthUsecase(authRepository)

	projectRepository := project.NewRepository(db)
	projectUsecase := project.NewUsecase(projectRepository)
	projectHandler := project.NewHandler(projectUsecase)

	roleRepository := role.NewRepository(db)
	roleUsecase := role.NewUsecase(roleRepository)
	roleHandler := role.NewHandler(roleUsecase)

	memberRepository := member.NewRepository(db)
	memberUsecase := member.NewUsecase(memberRepository)
	memberHandler := member.NewHandler(memberUsecase)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Server is running",
		})
	})

	api := router.Group("/api/v1")

	public := api.Group("")
	protected := api.Group("")
	protected.Use(authMiddleware.RequireToken())

	auth.NewAuthHandler(public, protected, authUsecase, cfg)

	projectHandler.RegisterRoutes(protected)
	roleHandler.RegisterRoutes(protected)
	memberHandler.RegisterRoutes(protected)

	slog.Info("Server started", "port", cfg.AppPort)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		return fmt.Errorf("run API server: %w", err)
	}

	return nil
}
