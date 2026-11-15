package application

import (
	"context"
	"errors"
	"time"

	"github.com/fickysp/project-hub-backend.git/internal/domain/model"
	"github.com/fickysp/project-hub-backend.git/internal/websocket"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ApplicationUsecase interface {
	SubmitApplication(ctx context.Context, projectID, applicantID uuid.UUID, req SubmitApplicationReq) error
	ReviewApplication(ctx context.Context, appID, reviewerID uuid.UUID, req ReviewApplicationReq) error
	CancelApplication(ctx context.Context, appID, applicantID uuid.UUID) error
	GetApplication(ctx context.Context, appID, requesterID uuid.UUID) (*ApplicationResponse, error)
	GetApplicationsByProject(ctx context.Context, projectID, requesterID uuid.UUID) ([]ApplicationResponse, error)
	GetApplicationsByApplicant(ctx context.Context, applicantID uuid.UUID) ([]ApplicationResponse, error)
	GetApplicationsByPosition(ctx context.Context, projectID, positionID, requesterID uuid.UUID) ([]ApplicationResponse, error)
}

type applicationUsecase struct {
	repo ApplicationRepository
	ws   websocket.Hub
}

func NewApplicationUsecase(repo ApplicationRepository, ws websocket.Hub) ApplicationUsecase {
	return &applicationUsecase{repo: repo, ws: ws}
}

func (u *applicationUsecase) SubmitApplication(ctx context.Context, projectID, applicantID uuid.UUID, req SubmitApplicationReq) error {
	exists, err := u.repo.CheckPositionExists(ctx, req.PositionID, projectID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("posisi tidak ditemukan pada project ini")
	}

	available, err := u.repo.CheckPositionAvailable(ctx, req.PositionID)
	if err != nil {
		return err
	}
	if !available {
		return errors.New("posisi sudah penuh atau tidak lagi menerima pengajuan")
	}

	hasActive, err := u.repo.CheckHasActiveApplication(ctx, applicantID, projectID)
	if err != nil {
		return err
	}
	if hasActive {
		return errors.New("kamu sudah memiliki pengajuan yang sedang diproses pada project ini")
	}

	app := &model.Application{
		ProjectID:      projectID,
		PositionID:     req.PositionID,
		ApplicantID:    applicantID,
		RelevantSkills: pq.StringArray(req.RelevantSkills),
		PortfolioUrl:   req.PortfolioURL,
		Motivation:     req.Motivation,
		Status:         model.AppStatusSubmitted,
	}

	if err := u.repo.Create(ctx, app); err != nil {
		return err
	}

	u.ws.BroadcastToProject(projectID, "APPLICATION_SUBMITTED", ApplicationWsResponse{
		ID:          app.ID,
		ProjectID:   app.ProjectID,
		PositionID:  app.PositionID,
		ApplicantID: app.ApplicantID,
		Status:      string(app.Status),
	})

	return nil
}

func (u *applicationUsecase) ReviewApplication(ctx context.Context, appID, reviewerID uuid.UUID, req ReviewApplicationReq) error {
	app, err := u.repo.FindByID(ctx, appID)
	if err != nil {
		return err
	}

	isLead, err := u.repo.CheckIsProjectLead(ctx, app.ProjectID, reviewerID)
	if err != nil {
		return err
	}
	if !isLead {
		return errors.New("forbidden: hanya lead project yang dapat melakukan aksi ini")
	}

	if app.Status == model.AppStatusAccepted ||
		app.Status == model.AppStatusRejected ||
		app.Status == model.AppStatusCancelled {
		return errors.New("pengajuan sudah diputuskan dan tidak dapat diubah lagi")
	}

	if req.Status == string(model.AppStatusRejected) &&
		(req.SelectionNote == nil || len(*req.SelectionNote) == 0) {
		return errors.New("catatan seleksi wajib diisi saat menolak pengajuan")
	}

	updateData := map[string]interface{}{
		"status":     req.Status,
		"updated_at": time.Now(),
	}

	if req.Status == string(model.AppStatusAccepted) ||
		req.Status == string(model.AppStatusRejected) {
		now := time.Now()
		updateData["decided_by"] = reviewerID
		updateData["decided_at"] = now
	}

	if req.SelectionNote != nil {
		updateData["selection_note"] = req.SelectionNote
	}

	if err := u.repo.UpdateStatus(ctx, appID, updateData); err != nil {
		return err
	}

	u.ws.BroadcastToProject(app.ProjectID, "APPLICATION_REVIEWED", ApplicationWsResponse{
		ID:          app.ID,
		ProjectID:   app.ProjectID,
		PositionID:  app.PositionID,
		ApplicantID: app.ApplicantID,
		Status:      req.Status,
		DecidedBy:   &reviewerID,
	})

	return nil
}

func (u *applicationUsecase) CancelApplication(ctx context.Context, appID, applicantID uuid.UUID) error {
	app, err := u.repo.FindByID(ctx, appID)
	if err != nil {
		return err
	}

	if app.ApplicantID != applicantID {
		return errors.New("forbidden: kamu tidak memiliki akses untuk membatalkan pengajuan ini")
	}

	if app.Status != model.AppStatusSubmitted && app.Status != model.AppStatusReviewing {
		return errors.New("pengajuan sudah diputuskan dan tidak dapat dibatalkan")
	}

	updateData := map[string]interface{}{
		"status":     model.AppStatusCancelled,
		"updated_at": time.Now(),
	}

	if err := u.repo.UpdateStatus(ctx, appID, updateData); err != nil {
		return err
	}

	u.ws.BroadcastToProject(app.ProjectID, "APPLICATION_CANCELLED", ApplicationWsResponse{
		ID:          app.ID,
		ProjectID:   app.ProjectID,
		PositionID:  app.PositionID,
		ApplicantID: app.ApplicantID,
		Status:      string(model.AppStatusCancelled),
	})

	return nil
}

func (u *applicationUsecase) GetApplication(ctx context.Context, appID, requesterID uuid.UUID) (*ApplicationResponse, error) {
	app, err := u.repo.FindByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	isLead, err := u.repo.CheckIsProjectLead(ctx, app.ProjectID, requesterID)
	if err != nil {
		return nil, err
	}
	if app.ApplicantID != requesterID && !isLead {
		return nil, errors.New("forbidden: kamu tidak memiliki akses untuk melihat pengajuan ini")
	}

	response := toApplicationResponse(app)
	return &response, nil
}

func (u *applicationUsecase) GetApplicationsByProject(ctx context.Context, projectID, requesterID uuid.UUID) ([]ApplicationResponse, error) {
	isLead, err := u.repo.CheckIsProjectLead(ctx, projectID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isLead {
		return nil, errors.New("forbidden: hanya lead project yang dapat melihat semua pengajuan")
	}

	apps, err := u.repo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := make([]ApplicationResponse, 0, len(apps))
	for _, app := range apps {
		result = append(result, toApplicationResponse(&app))
	}
	return result, nil
}

func (u *applicationUsecase) GetApplicationsByApplicant(ctx context.Context, applicantID uuid.UUID) ([]ApplicationResponse, error) {
	apps, err := u.repo.FindByApplicantID(ctx, applicantID)
	if err != nil {
		return nil, err
	}

	result := make([]ApplicationResponse, 0, len(apps))
	for _, app := range apps {
		result = append(result, toApplicationResponse(&app))
	}
	return result, nil
}

func (u *applicationUsecase) GetApplicationsByPosition(ctx context.Context, projectID, positionID, requesterID uuid.UUID) ([]ApplicationResponse, error) {
	isLead, err := u.repo.CheckIsProjectLead(ctx, projectID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isLead {
		return nil, errors.New("forbidden: hanya lead project yang dapat melihat pengajuan per posisi")
	}

	apps, err := u.repo.FindByProjectAndPosition(ctx, projectID, positionID)
	if err != nil {
		return nil, err
	}

	result := make([]ApplicationResponse, 0, len(apps))
	for _, app := range apps {
		result = append(result, toApplicationResponse(&app))
	}
	return result, nil
}
