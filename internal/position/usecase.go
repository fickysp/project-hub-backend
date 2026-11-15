package projectposition

import (
	"context"
	"errors"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/fickysp/project-hub-backend.git/internal/websocket"
	"github.com/google/uuid"
)

type PositionUsecase interface {
	CreatePosition(ctx context.Context, projectID, userID uuid.UUID, req CreatePositionReq) error
	UpdatePosition(ctx context.Context, posID, userID uuid.UUID, req UpdatePositionReq) error
	DeletePosition(ctx context.Context, posID, userID uuid.UUID) error
	GetPosition(ctx context.Context, posID uuid.UUID) (*PositionResponse, error)
	GetAllPosition(ctx context.Context, projectID uuid.UUID) ([]PositionResponse, error)
}

type positionUsecase struct {
	repo PositionRepository
	ws   websocket.Hub
}

func NewPositionUsecase(repo PositionRepository, ws websocket.Hub) PositionUsecase {
	return &positionUsecase{repo: repo, ws: ws}
}

func (u *positionUsecase) CreatePosition(ctx context.Context, projectID, userID uuid.UUID, req CreatePositionReq) error {
	isLead, err := u.repo.CheckIsProjectLead(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !isLead {
		return errors.New("forbidden: hanya lead project yang dapat melakukan aksi ini")
	}

	pos := &model.ProjectPosition{
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
		Status:      model.PositionStatusOpen,
	}

	if err := u.repo.Create(ctx, pos); err != nil {
		return err
	}

	u.ws.BroadcastToProject(projectID, "POSITION_CREATED", PositionWsResponse{
		ID:        pos.ID,
		ProjectID: pos.ProjectID,
		Name:      pos.Name,
		Capacity:  pos.Capacity,
		Available: pos.Capacity,
		Status:    string(pos.Status),
	})

	return nil
}

func (u *positionUsecase) UpdatePosition(ctx context.Context, posID, userID uuid.UUID, req UpdatePositionReq) error {
	pos, err := u.repo.FindByID(ctx, posID)
	if err != nil {
		return err
	}

	isLead, err := u.repo.CheckIsProjectLead(ctx, pos.ProjectID, userID)
	if err != nil {
		return err
	}
	if !isLead {
		return errors.New("forbidden: hanya lead project yang dapat melakukan aksi ini")
	}

	updateData := make(map[string]interface{})

	if req.Capacity != nil {
		activeMembers, err := u.repo.GetActiveMembersCount(ctx, posID)
		if err != nil {
			return err
		}
		if *req.Capacity < int(activeMembers) {
			return errors.New("kapasitas tidak bisa lebih kecil dari jumlah anggota aktif saat ini")
		}
		updateData["capacity"] = *req.Capacity
		pos.Capacity = *req.Capacity
	}

	if req.Name != nil {
		updateData["name"] = *req.Name
		pos.Name = *req.Name
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
	}
	if req.Status != nil {
		updateData["status"] = *req.Status
		pos.Status = model.PositionStatus(*req.Status)
	}

	if len(updateData) > 0 {
		if err := u.repo.UpdatePartial(ctx, posID, updateData); err != nil {
			return err
		}

		activeMembers, _ := u.repo.GetActiveMembersCount(ctx, posID)
		u.ws.BroadcastToProject(pos.ProjectID, "POSITION_UPDATED", PositionWsResponse{
			ID:        pos.ID,
			ProjectID: pos.ProjectID,
			Name:      pos.Name,
			Capacity:  pos.Capacity,
			Available: pos.Capacity - int(activeMembers),
			Status:    string(pos.Status),
		})
	}

	return nil
}

func (u *positionUsecase) DeletePosition(ctx context.Context, posID, userID uuid.UUID) error {
	pos, err := u.repo.FindByID(ctx, posID)
	if err != nil {
		return err
	}

	isLead, err := u.repo.CheckIsProjectLead(ctx, pos.ProjectID, userID)
	if err != nil {
		return err
	}
	if !isLead {
		return errors.New("forbidden: hanya lead project yang dapat melakukan aksi ini")
	}

	activeMembers, err := u.repo.GetActiveMembersCount(ctx, posID)
	if err != nil {
		return err
	}
	if activeMembers > 0 {
		return errors.New("tidak dapat menghapus posisi: masih ada anggota aktif")
	}

	pendingApps, err := u.repo.GetPendingAppsCount(ctx, posID)
	if err != nil {
		return err
	}
	if pendingApps > 0 {
		return errors.New("tidak dapat menghapus posisi: masih ada pengajuan yang diproses")
	}

	if err := u.repo.Delete(ctx, posID); err != nil {
		return err
	}

	u.ws.BroadcastToProject(pos.ProjectID, "POSITION_DELETED", map[string]interface{}{
		"id":         posID,
		"project_id": pos.ProjectID,
	})

	return nil
}

func (u *positionUsecase) GetPosition(ctx context.Context, posID uuid.UUID) (*PositionResponse, error) {
	pos, err := u.repo.FindByID(ctx, posID)
	if err != nil {
		return nil, err
	}

	activeMembers, err := u.repo.GetActiveMembersCount(ctx, posID)
	if err != nil {
		return nil, err
	}

	return &PositionResponse{
		ID:          pos.ID,
		ProjectID:   pos.ProjectID,
		Name:        pos.Name,
		Description: pos.Description,
		Capacity:    pos.Capacity,
		Available:   pos.Capacity - int(activeMembers),
		Status:      string(pos.Status),
	}, nil
}

func (u *positionUsecase) GetAllPosition(ctx context.Context, projectID uuid.UUID) ([]PositionResponse, error) {
	positions, err := u.repo.FindAllByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var result []PositionResponse
	for _, pos := range positions {
		activeMembers, err := u.repo.GetActiveMembersCount(ctx, pos.ID)
		if err != nil {
			return nil, err
		}

		result = append(result, PositionResponse{
			ID:          pos.ID,
			ProjectID:   pos.ProjectID,
			Name:        pos.Name,
			Description: pos.Description,
			Capacity:    pos.Capacity,
			Available:   pos.Capacity - int(activeMembers),
			Status:      string(pos.Status),
		})
	}

	if result == nil {
		result = []PositionResponse{}
	}

	return result, nil
}
