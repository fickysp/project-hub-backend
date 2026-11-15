package auth

import "github.com/fickysp/project-hub-backend.git/internal/domain/model"

type RegisterRequest struct {
	Name     string `json:"name" binding:"validate,min=3"`
	Email    string `json:"email" binding:"validate,email"`
	Password string `json:"password" binding:"validate,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"validate,email"`
	Password string `json:"password" binding:"validate,min=8"`
}

type LoginResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"-"`
	Data         model.User `json:"data"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}
