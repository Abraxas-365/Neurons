package benefitsrv

import (
	"context"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/benefit"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
)

type BenefitService struct {
	repo        benefit.Repository
	classrooms  *classroomsrv.ClassroomService
	enrollments *enrollmentsrv.EnrollmentService
}

func NewBenefitService(
	repo benefit.Repository,
	classrooms *classroomsrv.ClassroomService,
	enrollments *enrollmentsrv.EnrollmentService,
) *BenefitService {
	return &BenefitService{repo: repo, classrooms: classrooms, enrollments: enrollments}
}

func (s *BenefitService) Create(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req benefit.CreateBenefitRequest,
) (*benefit.Benefit, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, benefit.ErrInvalidInput("name is required")
	}
	if req.Cost != nil && *req.Cost <= 0 {
		return nil, benefit.ErrInvalidInput("cost must be positive")
	}
	if req.MaxUses != nil && *req.MaxUses <= 0 {
		return nil, benefit.ErrInvalidInput("max_uses must be positive")
	}
	if req.MaxUsesPerStudent != nil && *req.MaxUsesPerStudent <= 0 {
		return nil, benefit.ErrInvalidInput("max_uses_per_student must be positive")
	}
	if req.AvailableFrom != nil && req.AvailableUntil != nil && req.AvailableUntil.Before(*req.AvailableFrom) {
		return nil, benefit.ErrInvalidInput("available_until must be after available_from")
	}

	scope := req.Scope
	if scope == "" {
		scope = benefit.ScopeIndividual
	}

	now := time.Now()
	e := &benefit.Benefit{
		ID:                kernel.NewBenefitID(uuid.NewString()),
		ClassroomID:       classroomID,
		Name:              name,
		Description:       req.Description,
		Icon:              req.Icon,
		Cost:              req.Cost,
		MaxUses:           req.MaxUses,
		MaxUsesPerStudent: req.MaxUsesPerStudent,
		RequiresApproval:  req.RequiresApproval,
		Scope:             scope,
		Conditions:        req.Conditions,
		AvailableFrom:     req.AvailableFrom,
		AvailableUntil:    req.AvailableUntil,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *BenefitService) List(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	activeOnly bool,
) ([]benefit.Benefit, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}
	return s.repo.ListByClassroom(ctx, classroomID, activeOnly)
}

// ListForStudent returns the catalog as the student sees it (HU-061): only
// active benefits, annotated with availability and affordability.
func (s *BenefitService) ListForStudent(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
) ([]benefit.StudentView, error) {
	e, _, err := s.enrollments.MyEnrollment(ctx, classroomID, userID)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListByClassroom(ctx, classroomID, true)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	views := make([]benefit.StudentView, 0, len(items))
	for i := range items {
		views = append(views, items[i].ToStudentView(e.Balance, now))
	}
	return views, nil
}

func (s *BenefitService) Update(
	ctx context.Context,
	id kernel.BenefitID,
	teacherID kernel.UserID,
	req benefit.UpdateBenefitRequest,
) (*benefit.Benefit, error) {
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
			return nil, benefit.ErrInvalidInput("name cannot be empty")
		}
		e.Name = name
	}
	if req.Description != nil {
		e.Description = req.Description
	}
	if req.Icon != nil {
		e.Icon = req.Icon
	}
	if req.Cost != nil {
		if *req.Cost <= 0 {
			return nil, benefit.ErrInvalidInput("cost must be positive")
		}
		e.Cost = req.Cost
	}
	if req.MaxUses != nil {
		e.MaxUses = req.MaxUses
	}
	if req.MaxUsesPerStudent != nil {
		e.MaxUsesPerStudent = req.MaxUsesPerStudent
	}
	if req.RequiresApproval != nil {
		e.RequiresApproval = *req.RequiresApproval
	}
	if req.Scope != nil {
		e.Scope = *req.Scope
	}
	if req.Conditions != nil {
		e.Conditions = req.Conditions
	}
	if req.AvailableFrom != nil {
		e.AvailableFrom = req.AvailableFrom
	}
	if req.AvailableUntil != nil {
		e.AvailableUntil = req.AvailableUntil
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

func (s *BenefitService) Delete(ctx context.Context, id kernel.BenefitID, teacherID kernel.UserID) error {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, e.ClassroomID, teacherID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// ResolveForRedemption validates that a benefit can be redeemed by a student
// right now and returns the neuron cost to charge. Called by the ledger before
// it opens the transaction (§7 flow C).
func (s *BenefitService) ResolveForRedemption(
	ctx context.Context,
	id kernel.BenefitID,
	e *enrollment.Enrollment,
	requestedAmount *int,
) (*benefit.Benefit, int, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	if b.ClassroomID != e.ClassroomID {
		// RN-01: benefits never cross classroom boundaries.
		return nil, 0, benefit.ErrNotFound()
	}
	if !b.IsAvailableAt(time.Now()) {
		if b.MaxUses != nil && b.UsesCount >= *b.MaxUses {
			return nil, 0, benefit.ErrQuotaExhausted()
		}
		return nil, 0, benefit.ErrUnavailable()
	}

	if b.MaxUsesPerStudent != nil {
		used, err := s.repo.CountUsesByStudent(ctx, id, e.ID)
		if err != nil {
			return nil, 0, err
		}
		if used >= *b.MaxUsesPerStudent {
			return nil, 0, benefit.ErrStudentQuotaExhausted()
		}
	}

	amount := 0
	switch {
	case b.HasFixedCost():
		amount = *b.Cost
	case requestedAmount == nil || *requestedAmount <= 0:
		// HU-063: free-amount benefits require the student to pick a value.
		return nil, 0, benefit.ErrInvalidInput("amount is required for this benefit")
	default:
		amount = *requestedAmount
	}

	return b, amount, nil
}
