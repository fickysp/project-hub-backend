package projectposition

import "github.com/google/uuid"

type CreatePositionReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Capacity    int    `json:"capacity" binding:"required,gt=0"`
}

type UpdatePositionReq struct {
	Name        *string `json:"name" binding:"omitempty"`
	Description *string `json:"description" binding:"omitempty"`
	Capacity    *int    `json:"capacity" binding:"omitempty,gt=0"`
	Status      *string `json:"status" binding:"omitempty,oneof=open closed"`
}

type PositionWsResponse struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
	Capacity  int       `json:"capacity"`
	Available int       `json:"available_quota"`
	Status    string    `json:"status"`
}

type PositionResponse struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Capacity    int       `json:"capacity"`
	Available   int       `json:"available_quota"`
	Status      string    `json:"status"`
}
