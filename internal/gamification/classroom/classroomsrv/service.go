package classroomsrv

import (
	"context"
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/classroom"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
)

// UserLookup resolves users by email so teachers can be added by address.
type UserLookup interface {
	FindIDByEmail(ctx context.Context, tenantID kernel.TenantID, email string) (kernel.UserID, error)
}

type ClassroomService struct {
	repo  classroom.Repository
	users UserLookup
}

func NewClassroomService(repo classroom.Repository, users UserLookup) *ClassroomService {
	return &ClassroomService{repo: repo, users: users}
}

// inviteAlphabet omits characters that are easy to confuse when typed by hand.
const inviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateInviteCode() (string, error) {
	var sb strings.Builder
	for range 8 {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteAlphabet))))
		if err != nil {
			return "", err
		}
		sb.WriteByte(inviteAlphabet[n.Int64()])
	}
	return sb.String(), nil
}

func (s *ClassroomService) Create(
	ctx context.Context,
	tenantID kernel.TenantID,
	userID kernel.UserID,
	req classroom.CreateClassroomRequest,
) (*classroom.Classroom, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, classroom.ErrInvalidInput("name is required")
	}
	if req.InitialNeurons < 0 {
		return nil, classroom.ErrInvalidInput("initial_neurons cannot be negative")
	}

	status := req.Status
	if status == "" {
		status = classroom.StatusActive
	}
	joinPolicy := req.JoinPolicy
	if joinPolicy == "" {
		joinPolicy = classroom.JoinAuto
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, classroom.ErrInvalidInput("could not generate invite code")
	}

	now := time.Now()
	entity := &classroom.Classroom{
		ID:                  kernel.NewClassroomID(uuid.NewString()),
		TenantID:            tenantID,
		Name:                name,
		Section:             req.Section,
		Term:                req.Term,
		Description:         req.Description,
		Icon:                req.Icon,
		InviteCode:          code,
		Status:              status,
		UnlimitedIssuance:   req.UnlimitedIssuance,
		VaultBalance:        req.InitialNeurons,
		JoinPolicy:          joinPolicy,
		VoidWindowSeconds:   valueOr(req.VoidWindowSeconds, 0),
		ReconfirmThreshold:  valueOr(req.ReconfirmThreshold, 10),
		AllowFreeRedemption: valueOr(req.AllowFreeRedemption, true),
		RankingPublic:       valueOr(req.RankingPublic, false),
		StartsAt:            req.StartsAt,
		EndsAt:              req.EndsAt,
		CreatedBy:           userID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.repo.Create(ctx, entity); err != nil {
		return nil, err
	}

	// The creator is always the owning teacher.
	owner := &classroom.ClassroomTeacher{
		ClassroomID: entity.ID,
		UserID:      userID,
		Role:        classroom.RoleOwner,
		AddedAt:     now,
	}
	if err := s.repo.AddTeacher(ctx, owner); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *ClassroomService) GetByID(ctx context.Context, id kernel.ClassroomID) (*classroom.Classroom, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ClassroomService) GetByInviteCode(ctx context.Context, code string) (*classroom.Classroom, error) {
	return s.repo.GetByInviteCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
}

func (s *ClassroomService) ListForTeacher(
	ctx context.Context,
	tenantID kernel.TenantID,
	userID kernel.UserID,
	opts kernel.PaginationOptions,
) (kernel.Paginated[classroom.Classroom], error) {
	return s.repo.ListForTeacher(ctx, tenantID, userID, opts)
}

// RequireTeacher loads the classroom and asserts the user teaches it.
// Every teacher-facing operation funnels through here.
func (s *ClassroomService) RequireTeacher(
	ctx context.Context,
	id kernel.ClassroomID,
	userID kernel.UserID,
) (*classroom.Classroom, *classroom.ClassroomTeacher, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	t, err := s.repo.GetTeacher(ctx, id, userID)
	if err != nil {
		return nil, nil, err
	}
	return c, t, nil
}

// RequireOwner asserts the user owns the classroom (stricter than RequireTeacher).
func (s *ClassroomService) RequireOwner(
	ctx context.Context,
	id kernel.ClassroomID,
	userID kernel.UserID,
) (*classroom.Classroom, error) {
	c, t, err := s.RequireTeacher(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if t.Role != classroom.RoleOwner {
		return nil, classroom.ErrNotOwner()
	}
	return c, nil
}

func (s *ClassroomService) Update(
	ctx context.Context,
	id kernel.ClassroomID,
	userID kernel.UserID,
	req classroom.UpdateClassroomRequest,
) (*classroom.Classroom, error) {
	c, err := s.RequireOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, classroom.ErrInvalidInput("name cannot be empty")
		}
		c.Name = name
	}
	assign(&c.Section, req.Section)
	assign(&c.Term, req.Term)
	assign(&c.Description, req.Description)
	assign(&c.Icon, req.Icon)
	if req.JoinPolicy != nil {
		c.JoinPolicy = *req.JoinPolicy
	}
	if req.VoidWindowSeconds != nil {
		c.VoidWindowSeconds = *req.VoidWindowSeconds
	}
	if req.ReconfirmThreshold != nil {
		c.ReconfirmThreshold = *req.ReconfirmThreshold
	}
	if req.AllowFreeRedemption != nil {
		c.AllowFreeRedemption = *req.AllowFreeRedemption
	}
	if req.RankingPublic != nil {
		c.RankingPublic = *req.RankingPublic
	}
	if req.StartsAt != nil {
		c.StartsAt = req.StartsAt
	}
	if req.EndsAt != nil {
		c.EndsAt = req.EndsAt
	}
	if req.Status != nil {
		c.Status = *req.Status
		// RN-17: record when the course was closed so reports can use the date.
		if c.Status == classroom.StatusClosed && c.ClosedAt == nil {
			now := time.Now()
			c.ClosedAt = &now
		}
		if c.Status == classroom.StatusActive {
			c.ClosedAt = nil
		}
	}

	c.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ClassroomService) AddTeacher(
	ctx context.Context,
	id kernel.ClassroomID,
	requesterID kernel.UserID,
	req classroom.AddTeacherRequest,
) (*classroom.ClassroomTeacher, error) {
	c, err := s.RequireOwner(ctx, id, requesterID)
	if err != nil {
		return nil, err
	}

	targetID, err := s.users.FindIDByEmail(ctx, c.TenantID, strings.TrimSpace(req.Email))
	if err != nil {
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = classroom.RoleAssistant
	}
	if role == classroom.RoleOwner {
		return nil, classroom.ErrInvalidInput("a classroom can only have one owner")
	}

	t := &classroom.ClassroomTeacher{
		ClassroomID:    id,
		UserID:         targetID,
		Role:           role,
		GrantAllowance: req.GrantAllowance,
		AddedAt:        time.Now(),
	}
	if err := s.repo.AddTeacher(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *ClassroomService) RemoveTeacher(
	ctx context.Context,
	id kernel.ClassroomID,
	requesterID kernel.UserID,
	targetID kernel.UserID,
) error {
	if _, err := s.RequireOwner(ctx, id, requesterID); err != nil {
		return err
	}
	return s.repo.RemoveTeacher(ctx, id, targetID)
}

func (s *ClassroomService) ListTeachers(
	ctx context.Context,
	id kernel.ClassroomID,
	requesterID kernel.UserID,
) ([]classroom.ClassroomTeacher, error) {
	if _, _, err := s.RequireTeacher(ctx, id, requesterID); err != nil {
		return nil, err
	}
	return s.repo.ListTeachers(ctx, id)
}

func (s *ClassroomService) Delete(ctx context.Context, id kernel.ClassroomID, userID kernel.UserID) error {
	if _, err := s.RequireOwner(ctx, id, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func valueOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

func assign[T any](dst **T, src *T) {
	if src != nil {
		*dst = src
	}
}
