package enrollmentsrv

import (
	"context"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/classroom"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
)

// UserLookup resolves emails to user ids when a teacher adds students.
type UserLookup interface {
	FindIDByEmail(ctx context.Context, tenantID kernel.TenantID, email string) (kernel.UserID, error)
}

type EnrollmentService struct {
	repo       enrollment.Repository
	classrooms *classroomsrv.ClassroomService
	users      UserLookup
}

func NewEnrollmentService(
	repo enrollment.Repository,
	classrooms *classroomsrv.ClassroomService,
	users UserLookup,
) *EnrollmentService {
	return &EnrollmentService{repo: repo, classrooms: classrooms, users: users}
}

// JoinByCode is the student-initiated path (HU-022). Depending on the
// classroom's join policy the student lands ACTIVE or PENDING.
func (s *EnrollmentService) JoinByCode(
	ctx context.Context,
	userID kernel.UserID,
	req enrollment.JoinByCodeRequest,
) (*enrollment.Enrollment, error) {
	c, err := s.classrooms.GetByInviteCode(ctx, req.InviteCode)
	if err != nil {
		return nil, err
	}
	if !c.AcceptsEnrollment() {
		return nil, classroom.ErrClassroomNotActive()
	}

	// Re-joining is idempotent: an existing membership is returned as-is.
	if existing, err := s.repo.GetByUserAndClassroom(ctx, c.ID, userID); err == nil {
		return existing, nil
	}

	status := enrollment.StatusActive
	if c.JoinPolicy == classroom.JoinApproval {
		status = enrollment.StatusPending
	}

	e := newEnrollment(c.ID, userID, status, req.StudentCode)
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// InviteStudents enrolls a batch of students by email (HU-020 / HU-021).
// Each row reports its own outcome so the UI can surface partial failures.
func (s *EnrollmentService) InviteStudents(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req enrollment.InviteStudentsRequest,
) ([]enrollment.InviteResult, error) {
	c, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID)
	if err != nil {
		return nil, err
	}

	results := make([]enrollment.InviteResult, 0, len(req.Students))
	for _, entry := range req.Students {
		email := strings.ToLower(strings.TrimSpace(entry.Email))
		if email == "" {
			results = append(results, enrollment.InviteResult{
				Email: entry.Email, Status: "ERROR", Detail: "email is required",
			})
			continue
		}

		userID, err := s.users.FindIDByEmail(ctx, c.TenantID, email)
		if err != nil {
			// The account does not exist yet; the teacher should send an IAM
			// invitation first so the student can register.
			results = append(results, enrollment.InviteResult{
				Email: email, Status: "ERROR", Detail: "no account exists for this email",
			})
			continue
		}

		if _, err := s.repo.GetByUserAndClassroom(ctx, classroomID, userID); err == nil {
			results = append(results, enrollment.InviteResult{
				Email: email, Status: "ALREADY_ENROLLED",
			})
			continue
		}

		e := newEnrollment(classroomID, userID, enrollment.StatusActive, entry.StudentCode)
		if err := s.repo.Create(ctx, e); err != nil {
			results = append(results, enrollment.InviteResult{
				Email: email, Status: "ERROR", Detail: err.Error(),
			})
			continue
		}
		results = append(results, enrollment.InviteResult{Email: email, Status: "ENROLLED"})
	}

	return results, nil
}

func newEnrollment(
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
	status enrollment.Status,
	studentCode *string,
) *enrollment.Enrollment {
	now := time.Now()
	return &enrollment.Enrollment{
		ID:          kernel.NewEnrollmentID(uuid.NewString()),
		ClassroomID: classroomID,
		UserID:      userID,
		Status:      status,
		StudentCode: studentCode,
		JoinedAt:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Roster is the teacher's student list (HU-023).
func (s *EnrollmentService) Roster(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	filter enrollment.RosterFilter,
	opts kernel.PaginationOptions,
) (kernel.Paginated[enrollment.Enrollment], error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return kernel.Paginated[enrollment.Enrollment]{}, err
	}
	return s.repo.Roster(ctx, classroomID, filter, opts)
}

// MyClassrooms lists the classrooms a student belongs to, with balances.
func (s *EnrollmentService) MyClassrooms(
	ctx context.Context,
	tenantID kernel.TenantID,
	userID kernel.UserID,
) ([]enrollment.MyEnrollment, error) {
	rows, err := s.repo.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]enrollment.MyEnrollment, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToMyEnrollment())
	}
	return out, nil
}

// MyEnrollment returns the caller's own membership in one classroom (HU-082),
// joined with the classroom it belongs to. The student's wallet names the course
// and has to know whether it still accepts movements (RN-17), neither of which
// lives on the enrollment row itself.
func (s *EnrollmentService) MyEnrollment(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
) (*enrollment.Enrollment, *classroom.Classroom, error) {
	e, err := s.repo.GetByUserAndClassroom(ctx, classroomID, userID)
	if err != nil {
		return nil, nil, err
	}
	c, err := s.classrooms.GetByID(ctx, classroomID)
	if err != nil {
		return nil, nil, err
	}
	return e, c, nil
}

// GetForTeacher loads one enrollment, asserting the caller teaches its classroom.
func (s *EnrollmentService) GetForTeacher(
	ctx context.Context,
	id kernel.EnrollmentID,
	teacherID kernel.UserID,
) (*enrollment.Enrollment, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, e.ClassroomID, teacherID); err != nil {
		return nil, err
	}
	return e, nil
}

// Approve activates a pending enrollment (HU-022 approval flow).
func (s *EnrollmentService) Approve(
	ctx context.Context,
	id kernel.EnrollmentID,
	teacherID kernel.UserID,
) (*enrollment.Enrollment, error) {
	e, err := s.GetForTeacher(ctx, id, teacherID)
	if err != nil {
		return nil, err
	}
	e.Status = enrollment.StatusActive
	e.LeftAt = nil
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Withdraw retires a student while preserving balance and history (HU-024, RN-16).
func (s *EnrollmentService) Withdraw(
	ctx context.Context,
	id kernel.EnrollmentID,
	teacherID kernel.UserID,
) (*enrollment.Enrollment, error) {
	e, err := s.GetForTeacher(ctx, id, teacherID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	e.Status = enrollment.StatusWithdrawn
	e.LeftAt = &now
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Update edits roster metadata (student code, status).
func (s *EnrollmentService) Update(
	ctx context.Context,
	id kernel.EnrollmentID,
	teacherID kernel.UserID,
	req enrollment.UpdateEnrollmentRequest,
) (*enrollment.Enrollment, error) {
	e, err := s.GetForTeacher(ctx, id, teacherID)
	if err != nil {
		return nil, err
	}
	if req.StudentCode != nil {
		e.StudentCode = req.StudentCode
	}
	if req.Status != nil {
		e.Status = *req.Status
		if *req.Status == enrollment.StatusWithdrawn && e.LeftAt == nil {
			now := time.Now()
			e.LeftAt = &now
		}
		if *req.Status == enrollment.StatusActive {
			e.LeftAt = nil
		}
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// CountActive is used by dashboards and team grant previews.
func (s *EnrollmentService) CountActive(ctx context.Context, classroomID kernel.ClassroomID) (int, error) {
	return s.repo.CountActive(ctx, classroomID)
}

// ListActiveByIDs validates a multi-student selection belongs to the classroom
// and is active (RN-01, RN-16). Callers must already have authorized the actor.
func (s *EnrollmentService) ListActiveByIDs(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	ids []kernel.EnrollmentID,
) ([]enrollment.Enrollment, error) {
	return s.repo.ListActiveByIDs(ctx, classroomID, ids)
}

// ListActiveByTeam returns the members a team grant pays out (RN-07).
func (s *EnrollmentService) ListActiveByTeam(
	ctx context.Context,
	teamID kernel.TeamID,
) ([]enrollment.Enrollment, error) {
	return s.repo.ListActiveByTeam(ctx, teamID)
}

// SetTeam assigns a student to a team, keeping the membership history.
func (s *EnrollmentService) SetTeam(
	ctx context.Context,
	id kernel.EnrollmentID,
	teacherID kernel.UserID,
	teamID *kernel.TeamID,
) error {
	if _, err := s.GetForTeacher(ctx, id, teacherID); err != nil {
		return err
	}
	return s.repo.SetTeam(ctx, id, teamID)
}
