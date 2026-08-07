package team

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// Team groups students inside one classroom.
type Team struct {
	ID          kernel.TeamID      `json:"id" db:"id"`
	ClassroomID kernel.ClassroomID `json:"classroom_id" db:"classroom_id"`
	Name        string             `json:"name" db:"name"`
	Description *string            `json:"description" db:"description"`
	Color       *string            `json:"color" db:"color"`
	Icon        *string            `json:"icon" db:"icon"`
	Status      Status             `json:"status" db:"status"`
	CreatedAt   time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" db:"updated_at"`

	// MemberCount is populated by list queries; it counts active members only,
	// which is what a team grant will actually pay out (RN-07).
	MemberCount int `json:"member_count" db:"member_count"`
}

// Member is a student belonging to a team.
type Member struct {
	EnrollmentID  kernel.EnrollmentID `json:"enrollment_id" db:"enrollment_id"`
	UserID        kernel.UserID       `json:"user_id" db:"user_id"`
	Name          string              `json:"name" db:"name"`
	Email         string              `json:"email" db:"email"`
	Balance       int64               `json:"balance" db:"balance"`
	IsCoordinator bool                `json:"is_coordinator" db:"is_coordinator"`
	JoinedAt      time.Time           `json:"joined_at" db:"joined_at"`
}

// --- Request DTOs ---

type CreateTeamRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
	Icon        *string `json:"icon"`
	// Optional initial members.
	EnrollmentIDs []kernel.EnrollmentID `json:"enrollment_ids"`
}

type UpdateTeamRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
	Icon        *string `json:"icon"`
	Status      *Status `json:"status"`
}

type SetMembersRequest struct {
	EnrollmentIDs []kernel.EnrollmentID `json:"enrollment_ids"`
}

// RandomizeRequest drives automatic team creation (HU-032).
type RandomizeRequest struct {
	// Exactly one of TeamCount / TeamSize must be provided.
	TeamCount *int `json:"team_count"`
	TeamSize  *int `json:"team_size"`
	// NamePrefix defaults to "Equipo".
	NamePrefix string `json:"name_prefix"`
	// KeepTogether lists groups of students that must share a team.
	KeepTogether [][]kernel.EnrollmentID `json:"keep_together"`
	// KeepApart lists groups of students that must not share a team.
	KeepApart [][]kernel.EnrollmentID `json:"keep_apart"`
	// Preview only computes the distribution without persisting it.
	Preview bool `json:"preview"`
}

// RandomizeResult is the preview/outcome of a random distribution.
type RandomizeResult struct {
	Teams     []RandomizedTeam `json:"teams"`
	Persisted bool             `json:"persisted"`
	// Unplaced lists students the constraints made impossible to place.
	Unplaced []Member `json:"unplaced"`
}

type RandomizedTeam struct {
	ID      *kernel.TeamID `json:"id,omitempty"`
	Name    string         `json:"name"`
	Members []Member       `json:"members"`
}

// --- Response DTOs ---

type TeamResponse struct {
	ID          kernel.TeamID `json:"id"`
	Name        string        `json:"name"`
	Description *string       `json:"description"`
	Color       *string       `json:"color"`
	Icon        *string       `json:"icon"`
	Status      Status        `json:"status"`
	MemberCount int           `json:"member_count"`
	CreatedAt   time.Time     `json:"created_at"`
}

func (t *Team) ToResponse() TeamResponse {
	return TeamResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Color:       t.Color,
		Icon:        t.Icon,
		Status:      t.Status,
		MemberCount: t.MemberCount,
		CreatedAt:   t.CreatedAt,
	}
}

// Detail includes the roster of the team, used by the team profile screen.
type Detail struct {
	TeamResponse
	Members []Member `json:"members"`
}
