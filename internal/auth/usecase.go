package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/fickysp/project-hub-backend.git/pkg/bcrypt"
	"github.com/fickysp/project-hub-backend.git/pkg/jwt"
	"github.com/fickysp/project-hub-backend.git/pkg/sentinel"
)

type AuthUsecase interface {
	Register(ctx context.Context, req RegisterRequest) error
	Login(ctx context.Context, jwtSecret, refreshSecret string, req LoginRequest) (*LoginResponse, error)
}
type authUsecase struct {
	authRepo AuthRepository
}

func NewAuthUsecase(ar AuthRepository) AuthUsecase {
	return &authUsecase{
		authRepo: ar,
	}
}

func (u *authUsecase) Register(ctx context.Context, req RegisterRequest) error {
	user, err := u.authRepo.FindByEmail(ctx, req.Email)
	if user != nil {
		slog.Warn("Gagal registrasi, email sudah digunakan", "error", err,
			slog.String("email", req.Email))
		return sentinel.ErrEmailIsUsed
	} else if err != nil {
		return sentinel.ErrInternalServerError
	}

	hashPassword, err := bcrypt.HashPassword(req.Password)
	if err != nil {
		slog.Warn("Gagal menghash password", "error", err,
			slog.String("email", req.Email))
		return sentinel.ErrInternalServerError
	}

	users := &model.User{
		Name:      req.Name,
		Email:     req.Email,
		Password:  hashPassword,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := u.authRepo.Create(ctx, users); err != nil {
		slog.Warn("Gagal menambahkan user", "error", err,
			slog.String("email", req.Email))
		return sentinel.ErrInternalServerError
	}
	return nil
}

func (u *authUsecase) Login(ctx context.Context, jwtSecret, refreshSecret string, req LoginRequest) (*LoginResponse, error) {
	user, err := u.authRepo.FindByEmail(ctx, req.Email)
	if user == nil && err != nil {
		return nil, sentinel.ErrInternalServerError
	} else if user == nil && err == nil {
		slog.Warn("login failed: user not found", "email", req.Email)
		return nil, sentinel.ErrUserNotFound
	}

	if !bcrypt.CheckPassword(user.Password, req.Password) {
		return nil, sentinel.ErrInvalidCredentials
	}

	accessToken, err := jwt.GenerateAccessToken(user.ID.String(), user.Name, jwtSecret, user.UpdatedAt)
	if err != nil {
		slog.Warn("Gagal membuat access token: ", "error", err,
			slog.String("email", req.Email))
		return nil, sentinel.ErrInternalServerError
	}

	refreshToken, err := jwt.GenerateRefreshToken(user.ID.String(), user.Name, refreshSecret, user.UpdatedAt)
	if err != nil {
		slog.Warn("Gagal membuat refresh token: ", "error", err,
			slog.String("email", req.Email))
		return nil, sentinel.ErrInternalServerError
	}

	if err := u.authRepo.UpdateRefreshToken(ctx, user.ID.String(), refreshToken); err != nil {
		slog.Error("Gagal mengupdate refresh token", "error", err,
			slog.String("user_id", user.ID.String()))
		return nil, err
	}

	return &LoginResponse{
		AccessToken: accessToken,
		Data:        *user,
	}, nil
}
