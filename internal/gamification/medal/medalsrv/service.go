package medalsrv

import (
	"context"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/medal"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
)

type MedalService struct {
	repo        medal.Repository
	classrooms  *classroomsrv.ClassroomService
	enrollments *enrollmentsrv.EnrollmentService
}

func NewMedalService(
	repo medal.Repository,
	classrooms *classroomsrv.ClassroomService,
	enrollments *enrollmentsrv.EnrollmentService,
) *MedalService {
	return &MedalService{repo: repo, classrooms: classrooms, enrollments: enrollments}
}

func boolOr(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func (s *MedalService) Create(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req medal.CreateMedalRequest,
) (*medal.Medal, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, medal.ErrInvalidInput("name is required")
	}
	if req.MaxAwards != nil && *req.MaxAwards <= 0 {
		return nil, medal.ErrInvalidInput("max_awards must be positive")
	}

	medalType := req.Type
	if medalType == "" {
		medalType = medal.TypeIndividual
	}
	if medalType != medal.TypeIndividual && medalType != medal.TypeTeam {
		return nil, medal.ErrInvalidInput("type must be INDIVIDUAL or TEAM")
	}

	now := time.Now()
	e := &medal.Medal{
		ID:                  kernel.NewMedalID(uuid.NewString()),
		ClassroomID:         classroomID,
		Name:                name,
		Description:         req.Description,
		ImageURL:            req.ImageURL,
		Icon:                req.Icon,
		Category:            req.Category,
		Type:                medalType,
		Condition:           req.Condition,
		MaxAwards:           req.MaxAwards,
		Repeatable:          boolOr(req.Repeatable, true),
		ShowOnMemberProfile: boolOr(req.ShowOnMemberProfile, true),
		VisibleToStudents:   boolOr(req.VisibleToStudents, true),
		AvailableFrom:       req.AvailableFrom,
		AvailableUntil:      req.AvailableUntil,
		IsActive:            true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *MedalService) List(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	activeOnly bool,
) ([]medal.Medal, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}
	return s.repo.ListByClassroom(ctx, classroomID, activeOnly)
}

func (s *MedalService) Update(
	ctx context.Context,
	id kernel.MedalID,
	teacherID kernel.UserID,
	req medal.UpdateMedalRequest,
) (*medal.Medal, error) {
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
			return nil, medal.ErrInvalidInput("name cannot be empty")
		}
		e.Name = name
	}
	if req.Description != nil {
		e.Description = req.Description
	}
	if req.ImageURL != nil {
		e.ImageURL = req.ImageURL
	}
	if req.Icon != nil {
		e.Icon = req.Icon
	}
	if req.Category != nil {
		e.Category = req.Category
	}
	if req.Type != nil {
		e.Type = *req.Type
	}
	if req.Condition != nil {
		e.Condition = req.Condition
	}
	if req.MaxAwards != nil {
		e.MaxAwards = req.MaxAwards
	}
	if req.Repeatable != nil {
		e.Repeatable = *req.Repeatable
	}
	if req.ShowOnMemberProfile != nil {
		e.ShowOnMemberProfile = *req.ShowOnMemberProfile
	}
	if req.VisibleToStudents != nil {
		e.VisibleToStudents = *req.VisibleToStudents
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

func (s *MedalService) Delete(ctx context.Context, id kernel.MedalID, teacherID kernel.UserID) error {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, e.ClassroomID, teacherID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// Award grants a medal to one or more students, or to a team (HU-072, HU-073).
// Medals carry no neuron value, so no ledger entry is written.
func (s *MedalService) Award(
	ctx context.Context,
	id kernel.MedalID,
	teacherID kernel.UserID,
	req medal.AwardRequest,
) ([]medal.Award, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c, _, err := s.classrooms.RequireTeacher(ctx, m.ClassroomID, teacherID)
	if err != nil {
		return nil, err
	}
	// RN-17: a closed classroom accepts no new recognitions.
	if !c.AcceptsTransactions() {
		return nil, medal.ErrUnavailable()
	}
	if !m.IsAvailableAt(time.Now()) {
		return nil, medal.ErrUnavailable()
	}

	if req.TeamID != nil && len(req.EnrollmentIDs) > 0 {
		return nil, medal.ErrInvalidInput("provide either team_id or enrollment_ids, not both")
	}
	if req.TeamID == nil && len(req.EnrollmentIDs) == 0 {
		return nil, medal.ErrInvalidInput("no recipients provided")
	}
	if req.TeamID != nil && m.Type != medal.TypeTeam {
		return nil, medal.ErrWrongType()
	}
	if len(req.EnrollmentIDs) > 0 && m.Type != medal.TypeIndividual {
		return nil, medal.ErrWrongType()
	}

	targets, err := s.resolveTargets(ctx, m, req)
	if err != nil {
		return nil, err
	}

	if m.MaxAwards != nil {
		awarded, err := s.repo.CountAwards(ctx, id)
		if err != nil {
			return nil, err
		}
		if awarded+len(targets) > *m.MaxAwards {
			return nil, medal.ErrQuotaExhausted()
		}
	}

	now := time.Now()
	awards := make([]medal.Award, 0, len(targets))
	for _, t := range targets {
		if !m.Repeatable {
			exists, err := s.repo.HasAward(ctx, id, t.enrollmentID, t.teamID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, medal.ErrAlreadyAwarded()
			}
		}
		awards = append(awards, medal.Award{
			ID:           uuid.NewString(),
			MedalID:      id,
			ClassroomID:  m.ClassroomID,
			EnrollmentID: t.enrollmentID,
			TeamID:       t.teamID,
			AwardedBy:    teacherID,
			Note:         req.Note,
			AwardedAt:    now,
		})
	}

	if err := s.repo.AwardMany(ctx, awards); err != nil {
		return nil, err
	}
	return awards, nil
}

type awardTarget struct {
	enrollmentID *kernel.EnrollmentID
	teamID       *kernel.TeamID
}

func (s *MedalService) resolveTargets(
	ctx context.Context,
	m *medal.Medal,
	req medal.AwardRequest,
) ([]awardTarget, error) {
	if req.TeamID != nil {
		return []awardTarget{{teamID: req.TeamID}}, nil
	}

	// RN-01: recipients must belong to this classroom and be active.
	active, err := s.enrollments.ListActiveByIDs(ctx, m.ClassroomID, req.EnrollmentIDs)
	if err != nil {
		return nil, err
	}
	if len(active) != len(req.EnrollmentIDs) {
		return nil, medal.ErrInvalidInput("some recipients are not active students of this classroom")
	}

	targets := make([]awardTarget, 0, len(active))
	for i := range active {
		id := active[i].ID
		targets = append(targets, awardTarget{enrollmentID: &id})
	}
	return targets, nil
}

// Revoke soft-deletes an award; the row stays for auditing (RN-15).
func (s *MedalService) Revoke(ctx context.Context, awardID string, teacherID kernel.UserID) error {
	a, err := s.repo.GetAward(ctx, awardID)
	if err != nil {
		return err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, a.ClassroomID, teacherID); err != nil {
		return err
	}
	return s.repo.RevokeAward(ctx, awardID)
}

func (s *MedalService) ClassroomAwards(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
) ([]medal.Award, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}
	return s.repo.ListAwardsByClassroom(ctx, classroomID)
}

// MyMedals returns the student's own medal wall (HU-074).
func (s *MedalService) MyMedals(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
) ([]medal.Award, error) {
	e, _, err := s.enrollments.MyEnrollment(ctx, classroomID, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAwardsForStudent(ctx, e.ID)
}

// StudentAwards returns one student's medals for the teacher-facing profile.
func (s *MedalService) StudentAwards(
	ctx context.Context,
	e *enrollment.Enrollment,
) ([]medal.Award, error) {
	return s.repo.ListAwardsForStudent(ctx, e.ID)
}
