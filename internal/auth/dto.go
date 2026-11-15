package auth

import "github.com/fickysp/project-hub-backend.git/internal/domain/model"

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"-"`
	Data         model.User `json:"data"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}
