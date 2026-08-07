package benefit

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

type Scope string

const (
	ScopeIndividual Scope = "INDIVIDUAL"
	ScopeTeam       Scope = "TEAM"
)

// Benefit is what a student can obtain by returning neurons (HU-041).
type Benefit struct {
	ID          kernel.BenefitID   `json:"id" db:"id"`
	ClassroomID kernel.ClassroomID `json:"classroom_id" db:"classroom_id"`
	Name        string             `json:"name" db:"name"`
	Description *string            `json:"description" db:"description"`
	Icon        *string            `json:"icon" db:"icon"`
	// Cost is nil for free-amount benefits: the student chooses how much (HU-063).
	Cost *int `json:"cost" db:"cost"`
	// MaxUses caps redemptions across the whole classroom.
	MaxUses   *int `json:"max_uses" db:"max_uses"`
	UsesCount int  `json:"uses_count" db:"uses_count"`
	// MaxUsesPerStudent caps redemptions per student.
	MaxUsesPerStudent *int       `json:"max_uses_per_student" db:"max_uses_per_student"`
	RequiresApproval  bool       `json:"requires_approval" db:"requires_approval"`
	Scope             Scope      `json:"scope" db:"scope"`
	Conditions        *string    `json:"conditions" db:"conditions"`
	AvailableFrom     *time.Time `json:"available_from" db:"available_from"`
	AvailableUntil    *time.Time `json:"available_until" db:"available_until"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// HasFixedCost reports whether the price is set by the teacher.
func (b *Benefit) HasFixedCost() bool {
	return b.Cost != nil
}

// IsAvailableAt reports whether the benefit can be redeemed at the given time:
// active, inside its window and below its global quota.
func (b *Benefit) IsAvailableAt(now time.Time) bool {
	if !b.IsActive {
		return false
	}
	if b.AvailableFrom != nil && now.Before(*b.AvailableFrom) {
		return false
	}
	if b.AvailableUntil != nil && now.After(*b.AvailableUntil) {
		return false
	}
	if b.MaxUses != nil && b.UsesCount >= *b.MaxUses {
		return false
	}
	return true
}

// RemainingUses returns how many redemptions are still allowed globally.
func (b *Benefit) RemainingUses() *int {
	if b.MaxUses == nil {
		return nil
	}
	remaining := *b.MaxUses - b.UsesCount
	if remaining < 0 {
		remaining = 0
	}
	return &remaining
}

// --- Request DTOs ---

type CreateBenefitRequest struct {
	Name              string     `json:"name"`
	Description       *string    `json:"description"`
	Icon              *string    `json:"icon"`
	Cost              *int       `json:"cost"`
	MaxUses           *int       `json:"max_uses"`
	MaxUsesPerStudent *int       `json:"max_uses_per_student"`
	RequiresApproval  bool       `json:"requires_approval"`
	Scope             Scope      `json:"scope"`
	Conditions        *string    `json:"conditions"`
	AvailableFrom     *time.Time `json:"available_from"`
	AvailableUntil    *time.Time `json:"available_until"`
}

type UpdateBenefitRequest struct {
	Name              *string    `json:"name"`
	Description       *string    `json:"description"`
	Icon              *string    `json:"icon"`
	Cost              *int       `json:"cost"`
	MaxUses           *int       `json:"max_uses"`
	MaxUsesPerStudent *int       `json:"max_uses_per_student"`
	RequiresApproval  *bool      `json:"requires_approval"`
	Scope             *Scope     `json:"scope"`
	Conditions        *string    `json:"conditions"`
	AvailableFrom     *time.Time `json:"available_from"`
	AvailableUntil    *time.Time `json:"available_until"`
	IsActive          *bool      `json:"is_active"`
}

// --- Response DTOs ---

// StudentView is the catalog entry as a student sees it, including whether
// they can currently afford and redeem it.
type StudentView struct {
	ID               kernel.BenefitID `json:"id"`
	Name             string           `json:"name"`
	Description      *string          `json:"description"`
	Icon             *string          `json:"icon"`
	Cost             *int             `json:"cost"`
	RequiresApproval bool             `json:"requires_approval"`
	Conditions       *string          `json:"conditions"`
	RemainingUses    *int             `json:"remaining_uses"`
	Available        bool             `json:"available"`
	Affordable       bool             `json:"affordable"`
	AvailableUntil   *time.Time       `json:"available_until"`
}

func (b *Benefit) ToStudentView(balance int64, now time.Time) StudentView {
	affordable := true
	if b.Cost != nil {
		affordable = balance >= int64(*b.Cost)
	}
	return StudentView{
		ID:               b.ID,
		Name:             b.Name,
		Description:      b.Description,
		Icon:             b.Icon,
		Cost:             b.Cost,
		RequiresApproval: b.RequiresApproval,
		Conditions:       b.Conditions,
		RemainingUses:    b.RemainingUses(),
		Available:        b.IsAvailableAt(now),
		Affordable:       affordable,
		AvailableUntil:   b.AvailableUntil,
	}
}
