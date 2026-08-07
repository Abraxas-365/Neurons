package enrollment

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Status is the lifecycle of a student's membership in one classroom.
type Status string

const (
	// StatusPending means the student redeemed an invite code in an
	// approval-gated classroom and awaits the teacher's decision.
	StatusPending Status = "PENDING"
	StatusActive  Status = "ACTIVE"
	// StatusWithdrawn keeps the balance and history intact (RN-16).
	StatusWithdrawn Status = "WITHDRAWN"
)

// Enrollment is a student's membership in a classroom and holds that
// classroom's neuron balance. RN-01: balances never cross classrooms.
type Enrollment struct {
	ID          kernel.EnrollmentID `json:"id" db:"id"`
	ClassroomID kernel.ClassroomID  `json:"classroom_id" db:"classroom_id"`
	UserID      kernel.UserID       `json:"user_id" db:"user_id"`

	Balance       int64 `json:"balance" db:"balance"`
	TotalReceived int64 `json:"total_received" db:"total_received"`
	TotalReturned int64 `json:"total_returned" db:"total_returned"`

	TeamID      *kernel.TeamID `json:"team_id" db:"team_id"`
	Status      Status         `json:"status" db:"status"`
	StudentCode *string        `json:"student_code" db:"student_code"`

	JoinedAt       time.Time  `json:"joined_at" db:"joined_at"`
	LeftAt         *time.Time `json:"left_at" db:"left_at"`
	LastActivityAt *time.Time `json:"last_activity_at" db:"last_activity_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`

	// Joined from users for roster display.
	Name  string `json:"name,omitempty" db:"name"`
	Email string `json:"email,omitempty" db:"email"`
	// Joined from teams.
	TeamName *string `json:"team_name,omitempty" db:"team_name"`
	// Count of medals earned, populated by roster queries.
	MedalCount int `json:"medal_count" db:"medal_count"`
}

// CanTransact reports whether this student may send or receive neurons.
// Withdrawn and pending students are frozen (RN-16).
func (e *Enrollment) CanTransact() bool {
	return e.Status == StatusActive
}

// CanAfford implements RN-04: a student can never return more than they hold.
func (e *Enrollment) CanAfford(amount int64) bool {
	return e.Balance >= amount
}

// --- Request DTOs ---

// JoinByCodeRequest is the student-initiated path (HU-022).
type JoinByCodeRequest struct {
	InviteCode  string  `json:"invite_code"`
	StudentCode *string `json:"student_code"`
}

// InviteStudentsRequest adds students by email (HU-020 / HU-021 bulk upload).
type InviteStudentsRequest struct {
	Students []InviteStudentEntry `json:"students"`
}

type InviteStudentEntry struct {
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	StudentCode *string `json:"student_code"`
}

// InviteResult reports per-row outcomes so the UI can show errors before
// confirming a bulk import (HU-021).
type InviteResult struct {
	Email  string `json:"email"`
	Status string `json:"status"` // ENROLLED | ALREADY_ENROLLED | INVITED | ERROR
	Detail string `json:"detail,omitempty"`
}

type UpdateEnrollmentRequest struct {
	StudentCode *string `json:"student_code"`
	Status      *Status `json:"status"`
}

// RosterFilter drives the teacher's student list (HU-023).
type RosterFilter struct {
	Search string
	Status *Status
	TeamID *kernel.TeamID
	// SortBy accepts: name, balance, received, returned, activity
	SortBy string
	// SortDir accepts: asc, desc
	SortDir string
}

// --- Response DTOs ---

// RosterEntry is the teacher-facing row: full financial visibility.
type RosterEntry struct {
	ID             kernel.EnrollmentID `json:"id"`
	UserID         kernel.UserID       `json:"user_id"`
	Name           string              `json:"name"`
	Email          string              `json:"email"`
	StudentCode    *string             `json:"student_code"`
	TeamID         *kernel.TeamID      `json:"team_id"`
	TeamName       *string             `json:"team_name"`
	Balance        int64               `json:"balance"`
	TotalReceived  int64               `json:"total_received"`
	TotalReturned  int64               `json:"total_returned"`
	MedalCount     int                 `json:"medal_count"`
	Status         Status              `json:"status"`
	LastActivityAt *time.Time          `json:"last_activity_at"`
	JoinedAt       time.Time           `json:"joined_at"`
}

func (e *Enrollment) ToRosterEntry() RosterEntry {
	return RosterEntry{
		ID:             e.ID,
		UserID:         e.UserID,
		Name:           e.Name,
		Email:          e.Email,
		StudentCode:    e.StudentCode,
		TeamID:         e.TeamID,
		TeamName:       e.TeamName,
		Balance:        e.Balance,
		TotalReceived:  e.TotalReceived,
		TotalReturned:  e.TotalReturned,
		MedalCount:     e.MedalCount,
		Status:         e.Status,
		LastActivityAt: e.LastActivityAt,
		JoinedAt:       e.JoinedAt,
	}
}

// MyEnrollment is the student-facing view of their own membership. It never
// exposes other students' data (§10.4).
type MyEnrollment struct {
	ID            kernel.EnrollmentID `json:"id"`
	ClassroomID   kernel.ClassroomID  `json:"classroom_id"`
	ClassroomName string              `json:"classroom_name"`
	Section       *string             `json:"section"`
	Term          *string             `json:"term"`
	Icon          *string             `json:"icon"`
	Balance       int64               `json:"balance"`
	TotalReceived int64               `json:"total_received"`
	TotalReturned int64               `json:"total_returned"`
	TeamID        *kernel.TeamID      `json:"team_id"`
	TeamName      *string             `json:"team_name"`
	Status        Status              `json:"status"`
	ClassroomOpen bool                `json:"classroom_open"`
	JoinedAt      time.Time           `json:"joined_at"`
}

// MyEnrollmentRow is the raw join used to build MyEnrollment.
type MyEnrollmentRow struct {
	Enrollment
	ClassroomName   string  `db:"classroom_name"`
	Section         *string `db:"section"`
	Term            *string `db:"term"`
	Icon            *string `db:"icon"`
	ClassroomStatus string  `db:"classroom_status"`
}

func (r *MyEnrollmentRow) ToMyEnrollment() MyEnrollment {
	return MyEnrollment{
		ID:            r.ID,
		ClassroomID:   r.ClassroomID,
		ClassroomName: r.ClassroomName,
		Section:       r.Section,
		Term:          r.Term,
		Icon:          r.Icon,
		Balance:       r.Balance,
		TotalReceived: r.TotalReceived,
		TotalReturned: r.TotalReturned,
		TeamID:        r.TeamID,
		TeamName:      r.TeamName,
		Status:        r.Status,
		ClassroomOpen: r.ClassroomStatus == "ACTIVE",
		JoinedAt:      r.JoinedAt,
	}
}
