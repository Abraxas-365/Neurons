package teamsrv

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Abraxas-356/neurons/internal/gamification/classroom/classroomsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment/enrollmentsrv"
	"github.com/Abraxas-356/neurons/internal/gamification/team"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/google/uuid"
)

type TeamService struct {
	repo        team.Repository
	classrooms  *classroomsrv.ClassroomService
	enrollments *enrollmentsrv.EnrollmentService
	roster      enrollment.Repository
}

func NewTeamService(
	repo team.Repository,
	classrooms *classroomsrv.ClassroomService,
	enrollments *enrollmentsrv.EnrollmentService,
	roster enrollment.Repository,
) *TeamService {
	return &TeamService{repo: repo, classrooms: classrooms, enrollments: enrollments, roster: roster}
}

func (s *TeamService) Create(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req team.CreateTeamRequest,
) (*team.Team, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, team.ErrInvalidInput("name is required")
	}

	now := time.Now()
	t := &team.Team{
		ID:          kernel.NewTeamID(uuid.NewString()),
		ClassroomID: classroomID,
		Name:        name,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		Status:      team.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}

	for _, enrollmentID := range req.EnrollmentIDs {
		if err := s.enrollments.SetTeam(ctx, enrollmentID, teacherID, &t.ID); err != nil {
			return nil, err
		}
	}
	t.MemberCount = len(req.EnrollmentIDs)

	return t, nil
}

func (s *TeamService) ListByClassroom(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
) ([]team.Team, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}
	return s.repo.ListByClassroom(ctx, classroomID)
}

// GetForTeacher loads a team and asserts the caller teaches its classroom.
func (s *TeamService) GetForTeacher(
	ctx context.Context,
	id kernel.TeamID,
	teacherID kernel.UserID,
) (*team.Team, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.classrooms.RequireTeacher(ctx, t.ClassroomID, teacherID); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TeamService) Detail(ctx context.Context, id kernel.TeamID, teacherID kernel.UserID) (*team.Detail, error) {
	t, err := s.GetForTeacher(ctx, id, teacherID)
	if err != nil {
		return nil, err
	}
	members, err := s.repo.Members(ctx, id)
	if err != nil {
		return nil, err
	}
	return &team.Detail{TeamResponse: t.ToResponse(), Members: members}, nil
}

// MembersForStudent lets a student see their own team's composition (HU-073).
func (s *TeamService) MembersForStudent(
	ctx context.Context,
	id kernel.TeamID,
	userID kernel.UserID,
) (*team.Detail, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// The caller must belong to this team.
	e, _, err := s.enrollments.MyEnrollment(ctx, t.ClassroomID, userID)
	if err != nil {
		return nil, err
	}
	if e.TeamID == nil || *e.TeamID != id {
		return nil, enrollment.ErrForbidden()
	}

	members, err := s.repo.Members(ctx, id)
	if err != nil {
		return nil, err
	}
	return &team.Detail{TeamResponse: t.ToResponse(), Members: members}, nil
}

func (s *TeamService) Update(
	ctx context.Context,
	id kernel.TeamID,
	teacherID kernel.UserID,
	req team.UpdateTeamRequest,
) (*team.Team, error) {
	t, err := s.GetForTeacher(ctx, id, teacherID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, team.ErrInvalidInput("name cannot be empty")
		}
		t.Name = name
	}
	if req.Description != nil {
		t.Description = req.Description
	}
	if req.Color != nil {
		t.Color = req.Color
	}
	if req.Icon != nil {
		t.Icon = req.Icon
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	t.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// SetMembers replaces the team roster with the given students (HU-031).
func (s *TeamService) SetMembers(
	ctx context.Context,
	id kernel.TeamID,
	teacherID kernel.UserID,
	req team.SetMembersRequest,
) ([]team.Member, error) {
	if _, err := s.GetForTeacher(ctx, id, teacherID); err != nil {
		return nil, err
	}

	current, err := s.repo.Members(ctx, id)
	if err != nil {
		return nil, err
	}

	wanted := make(map[kernel.EnrollmentID]bool, len(req.EnrollmentIDs))
	for _, e := range req.EnrollmentIDs {
		wanted[e] = true
	}

	// Drop members no longer listed.
	for _, m := range current {
		if !wanted[m.EnrollmentID] {
			if err := s.enrollments.SetTeam(ctx, m.EnrollmentID, teacherID, nil); err != nil {
				return nil, err
			}
		}
	}

	// Add the newly listed ones. SetTeam closes any previous membership, which
	// enforces "one active team per student per classroom" (decision 15.10).
	existing := make(map[kernel.EnrollmentID]bool, len(current))
	for _, m := range current {
		existing[m.EnrollmentID] = true
	}
	for _, enrollmentID := range req.EnrollmentIDs {
		if !existing[enrollmentID] {
			if err := s.enrollments.SetTeam(ctx, enrollmentID, teacherID, &id); err != nil {
				return nil, err
			}
		}
	}

	return s.repo.Members(ctx, id)
}

func (s *TeamService) SetCoordinator(
	ctx context.Context,
	id kernel.TeamID,
	teacherID kernel.UserID,
	enrollmentID kernel.EnrollmentID,
) error {
	if _, err := s.GetForTeacher(ctx, id, teacherID); err != nil {
		return err
	}
	return s.repo.SetCoordinator(ctx, id, enrollmentID)
}

func (s *TeamService) Delete(ctx context.Context, id kernel.TeamID, teacherID kernel.UserID) error {
	if _, err := s.GetForTeacher(ctx, id, teacherID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// Randomize distributes active students into teams (HU-032), honouring
// keep-together and keep-apart constraints. With Preview set, nothing is saved.
func (s *TeamService) Randomize(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	teacherID kernel.UserID,
	req team.RandomizeRequest,
) (*team.RandomizeResult, error) {
	if _, _, err := s.classrooms.RequireTeacher(ctx, classroomID, teacherID); err != nil {
		return nil, err
	}

	students, err := s.roster.Roster(ctx, classroomID,
		enrollment.RosterFilter{Status: ptr(enrollment.StatusActive)},
		kernel.PaginationOptions{Page: 1, PageSize: 1000})
	if err != nil {
		return nil, err
	}
	if len(students.Items) == 0 {
		return nil, team.ErrNoStudents()
	}

	members := make([]team.Member, 0, len(students.Items))
	for i := range students.Items {
		e := &students.Items[i]
		members = append(members, team.Member{
			EnrollmentID: e.ID,
			UserID:       e.UserID,
			Name:         e.Name,
			Email:        e.Email,
			Balance:      e.Balance,
			JoinedAt:     e.JoinedAt,
		})
	}

	teamCount, err := resolveTeamCount(req, len(members))
	if err != nil {
		return nil, err
	}

	buckets, unplaced := distribute(members, teamCount, req.KeepTogether, req.KeepApart)

	prefix := strings.TrimSpace(req.NamePrefix)
	if prefix == "" {
		prefix = "Equipo"
	}

	result := &team.RandomizeResult{
		Teams:     make([]team.RandomizedTeam, 0, teamCount),
		Unplaced:  unplaced,
		Persisted: !req.Preview,
	}

	for i, bucket := range buckets {
		rt := team.RandomizedTeam{
			Name:    fmt.Sprintf("%s %d", prefix, i+1),
			Members: bucket,
		}

		if !req.Preview {
			created, err := s.Create(ctx, classroomID, teacherID, team.CreateTeamRequest{Name: rt.Name})
			if err != nil {
				return nil, err
			}
			for _, m := range bucket {
				if err := s.enrollments.SetTeam(ctx, m.EnrollmentID, teacherID, &created.ID); err != nil {
					return nil, err
				}
			}
			rt.ID = &created.ID
		}

		result.Teams = append(result.Teams, rt)
	}

	return result, nil
}

func resolveTeamCount(req team.RandomizeRequest, total int) (int, error) {
	switch {
	case req.TeamCount != nil && *req.TeamCount > 0:
		if *req.TeamCount > total {
			return total, nil
		}
		return *req.TeamCount, nil
	case req.TeamSize != nil && *req.TeamSize > 0:
		count := (total + *req.TeamSize - 1) / *req.TeamSize
		return count, nil
	default:
		return 0, team.ErrInvalidInput("provide either team_count or team_size")
	}
}

// distribute shuffles students and fills teams round-robin, keeping requested
// groups together and separating those that must not share a team. Constraints
// are best-effort: students that cannot be placed are reported back.
func distribute(
	members []team.Member,
	teamCount int,
	keepTogether [][]kernel.EnrollmentID,
	keepApart [][]kernel.EnrollmentID,
) ([][]team.Member, []team.Member) {
	byID := make(map[kernel.EnrollmentID]team.Member, len(members))
	for _, m := range members {
		byID[m.EnrollmentID] = m
	}

	// Group students that must stay together into single placement units.
	assigned := make(map[kernel.EnrollmentID]bool, len(members))
	units := [][]team.Member{}

	for _, group := range keepTogether {
		unit := []team.Member{}
		for _, id := range group {
			if m, ok := byID[id]; ok && !assigned[id] {
				unit = append(unit, m)
				assigned[id] = true
			}
		}
		if len(unit) > 0 {
			units = append(units, unit)
		}
	}
	for _, m := range members {
		if !assigned[m.EnrollmentID] {
			units = append(units, []team.Member{m})
			assigned[m.EnrollmentID] = true
		}
	}

	// Index the pairs that must not end up together.
	conflicts := make(map[kernel.EnrollmentID]map[kernel.EnrollmentID]bool)
	addConflict := func(a, b kernel.EnrollmentID) {
		if conflicts[a] == nil {
			conflicts[a] = map[kernel.EnrollmentID]bool{}
		}
		conflicts[a][b] = true
	}
	for _, group := range keepApart {
		for i := range group {
			for j := range group {
				if i != j {
					addConflict(group[i], group[j])
				}
			}
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(units), func(i, j int) { units[i], units[j] = units[j], units[i] })

	// Larger units first: they are the hardest to place.
	for i := 1; i < len(units); i++ {
		for j := i; j > 0 && len(units[j]) > len(units[j-1]); j-- {
			units[j], units[j-1] = units[j-1], units[j]
		}
	}

	buckets := make([][]team.Member, teamCount)
	unplaced := []team.Member{}

	fits := func(bucket []team.Member, unit []team.Member) bool {
		for _, existing := range bucket {
			for _, candidate := range unit {
				if conflicts[existing.EnrollmentID][candidate.EnrollmentID] {
					return false
				}
			}
		}
		return true
	}

	for _, unit := range units {
		// Prefer the smallest bucket that accepts the unit, keeping teams even.
		best := -1
		for i := range buckets {
			if !fits(buckets[i], unit) {
				continue
			}
			if best == -1 || len(buckets[i]) < len(buckets[best]) {
				best = i
			}
		}
		if best == -1 {
			unplaced = append(unplaced, unit...)
			continue
		}
		buckets[best] = append(buckets[best], unit...)
	}

	return buckets, unplaced
}

func ptr[T any](v T) *T { return &v }
