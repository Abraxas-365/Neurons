package classroom

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Status is the lifecycle state of a classroom.
type Status string

const (
	StatusDraft    Status = "DRAFT"
	StatusActive   Status = "ACTIVE"
	StatusClosed   Status = "CLOSED"
	StatusArchived Status = "ARCHIVED"
)

// JoinPolicy decides what happens when a student redeems an invite code.
type JoinPolicy string

const (
	JoinAuto     JoinPolicy = "AUTO"
	JoinApproval JoinPolicy = "APPROVAL"
)

// TeacherRole distinguishes the owner from assistants ("jefes de práctica").
type TeacherRole string

const (
	RoleOwner     TeacherRole = "OWNER"
	RoleAssistant TeacherRole = "ASSISTANT"
)

// Classroom is the aggregate root: an independent gamification universe with
// its own vault, catalogs, balances and ledger (RN-01).
type Classroom struct {
	ID       kernel.ClassroomID `json:"id" db:"id"`
	TenantID kernel.TenantID    `json:"tenant_id" db:"tenant_id"`

	Name        string  `json:"name" db:"name"`
	Section     *string `json:"section" db:"section"`
	Term        *string `json:"term" db:"term"`
	Description *string `json:"description" db:"description"`
	Icon        *string `json:"icon" db:"icon"`
	InviteCode  string  `json:"invite_code" db:"invite_code"`
	Status      Status  `json:"status" db:"status"`

	// Vault (bóveda). When UnlimitedIssuance is true, VaultBalance is not enforced.
	UnlimitedIssuance bool  `json:"unlimited_issuance" db:"unlimited_issuance"`
	VaultBalance      int64 `json:"vault_balance" db:"vault_balance"`
	TotalGranted      int64 `json:"total_granted" db:"total_granted"`
	TotalRedeemed     int64 `json:"total_redeemed" db:"total_redeemed"`

	JoinPolicy          JoinPolicy `json:"join_policy" db:"join_policy"`
	VoidWindowSeconds   int        `json:"void_window_seconds" db:"void_window_seconds"`
	ReconfirmThreshold  int        `json:"reconfirm_threshold" db:"reconfirm_threshold"`
	AllowFreeRedemption bool       `json:"allow_free_redemption" db:"allow_free_redemption"`
	RankingPublic       bool       `json:"ranking_public" db:"ranking_public"`

	StartsAt  *time.Time    `json:"starts_at" db:"starts_at"`
	EndsAt    *time.Time    `json:"ends_at" db:"ends_at"`
	ClosedAt  *time.Time    `json:"closed_at" db:"closed_at"`
	CreatedBy kernel.UserID `json:"created_by" db:"created_by"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" db:"updated_at"`
}

// AcceptsTransactions reports whether neurons may move in this classroom.
// RN-17: closed or archived classrooms are frozen; drafts are not yet live.
func (c *Classroom) AcceptsTransactions() bool {
	return c.Status == StatusActive
}

// AcceptsEnrollment reports whether students may still join.
func (c *Classroom) AcceptsEnrollment() bool {
	return c.Status == StatusActive
}

// CanGrant reports whether the vault can cover the given amount (RN-05).
func (c *Classroom) CanGrant(amount int64) bool {
	return c.UnlimitedIssuance || c.VaultBalance >= amount
}

// NeedsReconfirmation implements §11.9: large grants ask the teacher twice.
func (c *Classroom) NeedsReconfirmation(amount int64) bool {
	return c.ReconfirmThreshold > 0 && amount > int64(c.ReconfirmThreshold)
}

// ClassroomTeacher links a user to a classroom with teaching privileges.
type ClassroomTeacher struct {
	ClassroomID kernel.ClassroomID `json:"classroom_id" db:"classroom_id"`
	UserID      kernel.UserID      `json:"user_id" db:"user_id"`
	Role        TeacherRole        `json:"role" db:"role"`
	// GrantAllowance caps how many neurons an assistant may hand out. NULL = uncapped.
	GrantAllowance       *int64    `json:"grant_allowance" db:"grant_allowance"`
	GrantedFromAllowance int64     `json:"granted_from_allowance" db:"granted_from_allowance"`
	AddedAt              time.Time `json:"added_at" db:"added_at"`

	// Joined from users for display purposes.
	Name  string `json:"name,omitempty" db:"name"`
	Email string `json:"email,omitempty" db:"email"`
}

// RemainingAllowance returns how much this teacher may still grant.
func (t *ClassroomTeacher) RemainingAllowance() *int64 {
	if t.GrantAllowance == nil {
		return nil
	}
	remaining := *t.GrantAllowance - t.GrantedFromAllowance
	if remaining < 0 {
		remaining = 0
	}
	return &remaining
}

// --- Request DTOs ---

type CreateClassroomRequest struct {
	Name                string     `json:"name"`
	Section             *string    `json:"section"`
	Term                *string    `json:"term"`
	Description         *string    `json:"description"`
	Icon                *string    `json:"icon"`
	InitialNeurons      int64      `json:"initial_neurons"`
	UnlimitedIssuance   bool       `json:"unlimited_issuance"`
	JoinPolicy          JoinPolicy `json:"join_policy"`
	VoidWindowSeconds   *int       `json:"void_window_seconds"`
	ReconfirmThreshold  *int       `json:"reconfirm_threshold"`
	AllowFreeRedemption *bool      `json:"allow_free_redemption"`
	RankingPublic       *bool      `json:"ranking_public"`
	StartsAt            *time.Time `json:"starts_at"`
	EndsAt              *time.Time `json:"ends_at"`
	Status              Status     `json:"status"`
}

type UpdateClassroomRequest struct {
	Name                *string     `json:"name"`
	Section             *string     `json:"section"`
	Term                *string     `json:"term"`
	Description         *string     `json:"description"`
	Icon                *string     `json:"icon"`
	JoinPolicy          *JoinPolicy `json:"join_policy"`
	VoidWindowSeconds   *int        `json:"void_window_seconds"`
	ReconfirmThreshold  *int        `json:"reconfirm_threshold"`
	AllowFreeRedemption *bool       `json:"allow_free_redemption"`
	RankingPublic       *bool       `json:"ranking_public"`
	StartsAt            *time.Time  `json:"starts_at"`
	EndsAt              *time.Time  `json:"ends_at"`
	Status              *Status     `json:"status"`
}

type AddTeacherRequest struct {
	Email          string      `json:"email"`
	Role           TeacherRole `json:"role"`
	GrantAllowance *int64      `json:"grant_allowance"`
}

// --- Response DTOs ---

// ClassroomResponse is the teacher-facing view, including vault figures.
type ClassroomResponse struct {
	ID                  kernel.ClassroomID `json:"id"`
	Name                string             `json:"name"`
	Section             *string            `json:"section"`
	Term                *string            `json:"term"`
	Description         *string            `json:"description"`
	Icon                *string            `json:"icon"`
	InviteCode          string             `json:"invite_code"`
	Status              Status             `json:"status"`
	UnlimitedIssuance   bool               `json:"unlimited_issuance"`
	VaultBalance        int64              `json:"vault_balance"`
	TotalGranted        int64              `json:"total_granted"`
	TotalRedeemed       int64              `json:"total_redeemed"`
	Distributed         int64              `json:"distributed"`
	JoinPolicy          JoinPolicy         `json:"join_policy"`
	VoidWindowSeconds   int                `json:"void_window_seconds"`
	ReconfirmThreshold  int                `json:"reconfirm_threshold"`
	AllowFreeRedemption bool               `json:"allow_free_redemption"`
	RankingPublic       bool               `json:"ranking_public"`
	StartsAt            *time.Time         `json:"starts_at"`
	EndsAt              *time.Time         `json:"ends_at"`
	ClosedAt            *time.Time         `json:"closed_at"`
	CreatedAt           time.Time          `json:"created_at"`
}

func (c *Classroom) ToResponse() ClassroomResponse {
	return ClassroomResponse{
		ID:                  c.ID,
		Name:                c.Name,
		Section:             c.Section,
		Term:                c.Term,
		Description:         c.Description,
		Icon:                c.Icon,
		InviteCode:          c.InviteCode,
		Status:              c.Status,
		UnlimitedIssuance:   c.UnlimitedIssuance,
		VaultBalance:        c.VaultBalance,
		TotalGranted:        c.TotalGranted,
		TotalRedeemed:       c.TotalRedeemed,
		Distributed:         c.TotalGranted - c.TotalRedeemed,
		JoinPolicy:          c.JoinPolicy,
		VoidWindowSeconds:   c.VoidWindowSeconds,
		ReconfirmThreshold:  c.ReconfirmThreshold,
		AllowFreeRedemption: c.AllowFreeRedemption,
		RankingPublic:       c.RankingPublic,
		StartsAt:            c.StartsAt,
		EndsAt:              c.EndsAt,
		ClosedAt:            c.ClosedAt,
		CreatedAt:           c.CreatedAt,
	}
}

// StudentView hides vault internals from students (§10.4 privacy).
type StudentView struct {
	ID      kernel.ClassroomID `json:"id"`
	Name    string             `json:"name"`
	Section *string            `json:"section"`
	Term    *string            `json:"term"`
	Icon    *string            `json:"icon"`
	Status  Status             `json:"status"`
}

func (c *Classroom) ToStudentView() StudentView {
	return StudentView{
		ID:      c.ID,
		Name:    c.Name,
		Section: c.Section,
		Term:    c.Term,
		Icon:    c.Icon,
		Status:  c.Status,
	}
}
