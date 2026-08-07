package medal

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

type Type string

const (
	TypeIndividual Type = "INDIVIDUAL"
	TypeTeam       Type = "TEAM"
)

// Medal is a symbolic recognition inside a classroom (HU-071). Medals carry no
// neuron value; they never touch balances.
type Medal struct {
	ID          kernel.MedalID     `json:"id" db:"id"`
	ClassroomID kernel.ClassroomID `json:"classroom_id" db:"classroom_id"`
	Name        string             `json:"name" db:"name"`
	Description *string            `json:"description" db:"description"`
	ImageURL    *string            `json:"image_url" db:"image_url"`
	Icon        *string            `json:"icon" db:"icon"`
	Category    *string            `json:"category" db:"category"`
	Type        Type               `json:"type" db:"type"`
	Condition   *string            `json:"condition" db:"condition"`
	// MaxAwards caps how many times this medal may be granted in the classroom.
	MaxAwards *int `json:"max_awards" db:"max_awards"`
	// Repeatable allows awarding the same medal to the same target again (15.5).
	Repeatable bool `json:"repeatable" db:"repeatable"`
	// ShowOnMemberProfile makes a team medal visible on each member (RN-14).
	ShowOnMemberProfile bool       `json:"show_on_member_profile" db:"show_on_member_profile"`
	VisibleToStudents   bool       `json:"visible_to_students" db:"visible_to_students"`
	AvailableFrom       *time.Time `json:"available_from" db:"available_from"`
	AvailableUntil      *time.Time `json:"available_until" db:"available_until"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

func (m *Medal) IsAvailableAt(now time.Time) bool {
	if !m.IsActive {
		return false
	}
	if m.AvailableFrom != nil && now.Before(*m.AvailableFrom) {
		return false
	}
	if m.AvailableUntil != nil && now.After(*m.AvailableUntil) {
		return false
	}
	return true
}

// Award is one granted medal. Revocations are soft (RN-15): the row stays and
// RevokedAt is stamped.
type Award struct {
	ID          string             `json:"id" db:"id"`
	MedalID     kernel.MedalID     `json:"medal_id" db:"medal_id"`
	ClassroomID kernel.ClassroomID `json:"classroom_id" db:"classroom_id"`
	// Exactly one of EnrollmentID / TeamID is set.
	EnrollmentID *kernel.EnrollmentID `json:"enrollment_id" db:"enrollment_id"`
	TeamID       *kernel.TeamID       `json:"team_id" db:"team_id"`
	AwardedBy    kernel.UserID        `json:"awarded_by" db:"awarded_by"`
	Note         *string              `json:"note" db:"note"`
	AwardedAt    time.Time            `json:"awarded_at" db:"awarded_at"`
	RevokedAt    *time.Time           `json:"revoked_at" db:"revoked_at"`

	// Joined display fields.
	MedalName   *string `json:"medal_name,omitempty" db:"medal_name"`
	MedalIcon   *string `json:"medal_icon,omitempty" db:"medal_icon"`
	MedalImage  *string `json:"medal_image_url,omitempty" db:"medal_image_url"`
	StudentName *string `json:"student_name,omitempty" db:"student_name"`
	TeamName    *string `json:"team_name,omitempty" db:"team_name"`
}

func (a *Award) IsRevoked() bool { return a.RevokedAt != nil }

// --- Request DTOs ---

type CreateMedalRequest struct {
	Name                string     `json:"name"`
	Description         *string    `json:"description"`
	ImageURL            *string    `json:"image_url"`
	Icon                *string    `json:"icon"`
	Category            *string    `json:"category"`
	Type                Type       `json:"type"`
	Condition           *string    `json:"condition"`
	MaxAwards           *int       `json:"max_awards"`
	Repeatable          *bool      `json:"repeatable"`
	ShowOnMemberProfile *bool      `json:"show_on_member_profile"`
	VisibleToStudents   *bool      `json:"visible_to_students"`
	AvailableFrom       *time.Time `json:"available_from"`
	AvailableUntil      *time.Time `json:"available_until"`
}

type UpdateMedalRequest struct {
	Name                *string    `json:"name"`
	Description         *string    `json:"description"`
	ImageURL            *string    `json:"image_url"`
	Icon                *string    `json:"icon"`
	Category            *string    `json:"category"`
	Type                *Type      `json:"type"`
	Condition           *string    `json:"condition"`
	MaxAwards           *int       `json:"max_awards"`
	Repeatable          *bool      `json:"repeatable"`
	ShowOnMemberProfile *bool      `json:"show_on_member_profile"`
	VisibleToStudents   *bool      `json:"visible_to_students"`
	AvailableFrom       *time.Time `json:"available_from"`
	AvailableUntil      *time.Time `json:"available_until"`
	IsActive            *bool      `json:"is_active"`
}

// AwardRequest awards a medal to one or many students, or to a whole team
// (HU-072 / HU-073).
type AwardRequest struct {
	EnrollmentIDs []kernel.EnrollmentID `json:"enrollment_ids"`
	TeamID        *kernel.TeamID        `json:"team_id"`
	Note          *string               `json:"note"`
}
