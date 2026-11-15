package role

import (
	"context"
	"errors"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/google/uuid"
)

var (
	ErrUnauthorized    = errors.New("authentication required")
	ErrForbidden       = errors.New("only active project leads can change member roles")
	ErrProjectNotFound = errors.New("project not found")
	ErrMemberNotFound  = errors.New("project member not found")
	ErrMemberInactive  = errors.New("only active members can have their roles changed")
	ErrLastProjectLead = errors.New("the last project lead cannot be demoted")
	ErrInvalidID       = errors.New("project ID and member ID must not be empty UUIDs")
	ErrInvalidRole     = errors.New("role must be project_lead or member")
)

type Usecase interface {
	UpdateMemberRole(
		ctx context.Context,
		projectID uuid.UUID,
		memberID uuid.UUID,
		actorID uuid.UUID,
		request UpdateMemberRoleRequest,
	) (*MemberRoleResponse, error)
}

type usecase struct {
	repository Repository
}

func NewUsecase(repository Repository) Usecase {
	return &usecase{repository: repository}
}

func (u *usecase) UpdateMemberRole(
	ctx context.Context,
	projectID uuid.UUID,
	memberID uuid.UUID,
	actorID uuid.UUID,
	request UpdateMemberRoleRequest,
) (*MemberRoleResponse, error) {
	if actorID == uuid.Nil {
		return nil, ErrUnauthorized
	}

	if projectID == uuid.Nil || memberID == uuid.Nil {
		return nil, ErrInvalidID
	}

	switch request.Role {
	case model.ProjectRoleLead, model.ProjectRoleMember:
	default:
		return nil, ErrInvalidRole
	}

	member, err := u.repository.UpdateMemberRole(
		ctx,
		projectID,
		memberID,
		actorID,
		request.Role,
	)
	if err != nil {
		return nil, err
	}

	response := toMemberRoleResponse(member)

	return &response, nil
}
