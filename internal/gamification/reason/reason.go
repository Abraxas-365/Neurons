package reason

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Scope limits where a reason may be used.
type Scope string

const (
	ScopeIndividual Scope = "INDIVIDUAL"
	ScopeTeam       Scope = "TEAM"
	ScopeBoth       Scope = "BOTH"
)

// Reason is a preset justification for granting neurons (HU-040), shown as a
// large button on the quick-grant screen.
type Reason struct {
	ID          kernel.ReasonID    `json:"id" db:"id"`
	ClassroomID kernel.ClassroomID `json:"classroom_id" db:"classroom_id"`
	Name        string             `json:"name" db:"name"`
	Description *string            `json:"description" db:"description"`
	Icon        *string            `json:"icon" db:"icon"`
	// SuggestedAmount pre-selects a denomination in the UI.
	SuggestedAmount *int      `json:"suggested_amount" db:"suggested_amount"`
	Scope           Scope     `json:"scope" db:"scope"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// AppliesTo reports whether this reason may be used for the given target.
func (r *Reason) AppliesTo(s Scope) bool {
	return r.Scope == ScopeBoth || r.Scope == s
}

// --- Request DTOs ---

type CreateReasonRequest struct {
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	Icon            *string `json:"icon"`
	SuggestedAmount *int    `json:"suggested_amount"`
	Scope           Scope   `json:"scope"`
}

type UpdateReasonRequest struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	Icon            *string `json:"icon"`
	SuggestedAmount *int    `json:"suggested_amount"`
	Scope           *Scope  `json:"scope"`
	IsActive        *bool   `json:"is_active"`
}
