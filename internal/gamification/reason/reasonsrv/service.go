package reasonsrv

import (
	"context"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/reason"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
)

type ReasonService struct {
	repo       reason.Repository
	classrooms *classroomsrv.ClassroomService
}

func NewReasonService(repo reason.Repository, classrooms *classroomsrv.ClassroomService) *ReasonService {
	return &ReasonService{repo: repo, classrooms: classrooms}
}

func (s *ReasonService) Create(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req reason.CreateReasonRequest,
) (*reason.Reason, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, reason.ErrInvalidInput("name is required")
	}
	if req.SuggestedAmount != nil && *req.SuggestedAmount <= 0 {
		return nil, reason.ErrInvalidInput("suggested_amount must be positive")
	}

	scope := req.Scope
	if scope == "" {
		scope = reason.ScopeBoth
	}

	now := time.Now()
	e := &reason.Reason{
		ID:              kernel.NewReasonID(uuid.NewString()),
		ClassroomID:     classroomID,
		Name:            name,
		Description:     req.Description,
		Icon:            req.Icon,
		SuggestedAmount: req.SuggestedAmount,
		Scope:           scope,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *ReasonService) List(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	activeOnly bool,
) ([]reason.Reason, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}
	return s.repo.ListByClassroom(ctx, classroomID, activeOnly)
}

func (s *ReasonService) Update(
	ctx context.Context,
	id kernel.ReasonID,
	teacherID kernel.UserID,
	req reason.UpdateReasonRequest,
) (*reason.Reason, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, e.ClassroomID, teacherID); err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, reason.ErrInvalidInput("name cannot be empty")
		}
		e.Name = name
	}
	if req.Description != nil {
		e.Description = req.Description
	}
	if req.Icon != nil {
		e.Icon = req.Icon
	}
	if req.SuggestedAmount != nil {
		e.SuggestedAmount = req.SuggestedAmount
	}
	if req.Scope != nil {
		e.Scope = *req.Scope
	}
	if req.IsActive != nil {
		e.IsActive = *req.IsActive
	}
	e.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *ReasonService) Delete(ctx context.Context, id kernel.ReasonID, teacherID kernel.UserID) error {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, e.ClassroomID, teacherID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// GetForGrant validates that a catalog reason may back a grant: it must belong
// to the classroom (RN-01), still be active, and match the grant's scope.
// Authorization is the caller's responsibility.
func (s *ReasonService) GetForGrant(
	ctx context.Context,
	id kernel.ReasonID,
	classroomID kernel.ClassroomID,
	scope reason.Scope,
) (*reason.Reason, error) {
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.ClassroomID != classroomID {
		return nil, reason.ErrNotFound()
	}
	if !r.IsActive {
		return nil, reason.ErrInactive()
	}
	if !r.AppliesTo(scope) {
		return nil, reason.ErrWrongScope()
	}
	return r, nil
}
