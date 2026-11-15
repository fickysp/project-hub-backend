package member

import (
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
)

type ListMembersRequest struct {
	Page  int `form:"page,default=1" binding:"min=1,max=1000000"`
	Limit int `form:"limit,default=10" binding:"min=1,max=100"`
}

type UpdateMemberPositionRequest struct {
	PositionID uuid.UUID `json:"position_id" binding:"required"`
}

type MemberUserResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type MemberPositionResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type MemberResponse struct {
	ID        uuid.UUID               `json:"id"`
	ProjectID uuid.UUID               `json:"project_id"`
	User      MemberUserResponse      `json:"user"`
	Role      model.ProjectRole       `json:"role"`
	Status    model.MembershipStatus  `json:"status"`
	Position  *MemberPositionResponse `json:"position"`
	JoinedAt  time.Time               `json:"joined_at"`
}

type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type ListMembersResponse struct {
	Data       []MemberResponse   `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

type MembershipResponse struct {
	ID         uuid.UUID              `json:"id"`
	ProjectID  uuid.UUID              `json:"project_id"`
	UserID     uuid.UUID              `json:"user_id"`
	PositionID *uuid.UUID             `json:"position_id"`
	Role       model.ProjectRole      `json:"role"`
	Status     model.MembershipStatus `json:"status"`
	EndedAt    *time.Time             `json:"ended_at"`
}

func toMemberResponse(member *model.ProjectMember) MemberResponse {
	response := MemberResponse{
		ID:        member.ID,
		ProjectID: member.ProjectID,
		User: MemberUserResponse{
			ID:   member.UserID,
			Name: member.User.Name,
		},
		Role:     member.Role,
		Status:   member.Status,
		JoinedAt: member.JoinedAt,
	}

	if member.Position != nil {
		response.Position = &MemberPositionResponse{
			ID:   member.Position.ID,
			Name: member.Position.Name,
		}
	}

	return response
}

func toMembershipResponse(member *model.ProjectMember) MembershipResponse {
	return MembershipResponse{
		ID:         member.ID,
		ProjectID:  member.ProjectID,
		UserID:     member.UserID,
		PositionID: member.PositionID,
		Role:       member.Role,
		Status:     member.Status,
		EndedAt:    member.EndedAt,
	}
}
