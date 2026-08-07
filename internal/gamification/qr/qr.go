// Package qr implements the short-lived QR codes used to move neurons in class
// (flow A). Two directions exist:
//
//   - A student token identifies one enrollment so a teacher can scan it and
//     grant or receive neurons.
//   - A grant token encodes a prepared grant (amount + reason) that students
//     scan to claim.
//
// RN-12: every token is bound to one classroom and is useless anywhere else.
// RN-13: tokens are dynamic and expire in seconds, so a screenshot is worthless.
package qr

import (
	"time"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Kind distinguishes who displays the code.
type Kind string

const (
	// KindStudent is displayed by a student and scanned by the teacher.
	KindStudent Kind = "STUDENT"
	// KindGrant is displayed by the teacher and scanned by students.
	KindGrant Kind = "GRANT"
)

// DefaultStudentTTL keeps a student's identity code alive just long enough to
// be scanned (§11.2).
const DefaultStudentTTL = 60 * time.Second

// DefaultGrantTTL covers a short in-class activity.
const DefaultGrantTTL = 5 * time.Minute

// MaxGrantTTL caps how long a prepared grant may stay claimable.
const MaxGrantTTL = 30 * time.Minute

// Token is the payload stored in Redis behind an opaque code.
type Token struct {
	Code        string             `json:"code"`
	Kind        Kind               `json:"kind"`
	ClassroomID kernel.ClassroomID `json:"classroom_id"`

	// Student tokens
	EnrollmentID *kernel.EnrollmentID `json:"enrollment_id,omitempty"`
	UserID       *kernel.UserID       `json:"user_id,omitempty"`

	// Grant tokens
	Amount     int64            `json:"amount,omitempty"`
	ReasonID   *kernel.ReasonID `json:"reason_id,omitempty"`
	ReasonText *string          `json:"reason_text,omitempty"`
	IssuedBy   *kernel.UserID   `json:"issued_by,omitempty"`
	// MaxClaims caps how many students may claim one grant code. 0 = unlimited
	// while the code lives.
	MaxClaims int `json:"max_claims,omitempty"`

	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (t *Token) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

// TTL is what the client should use to schedule a refresh.
func (t *Token) TTL(now time.Time) time.Duration {
	d := t.ExpiresAt.Sub(now)
	if d < 0 {
		return 0
	}
	return d
}

// --- Request DTOs ---

// IssueGrantRequest prepares a scannable grant (HU-050).
type IssueGrantRequest struct {
	Amount     int64            `json:"amount"`
	ReasonID   *kernel.ReasonID `json:"reason_id"`
	ReasonText *string          `json:"reason_text"`
	TTLSeconds int              `json:"ttl_seconds"`
	MaxClaims  int              `json:"max_claims"`
}

// ScanRequest is what a teacher's scanner posts after decoding a student code.
type ScanRequest struct {
	Code string `json:"code"`
}

// ClaimRequest is what a student posts after scanning a teacher's grant code.
type ClaimRequest struct {
	Code string `json:"code"`
}

// --- Response DTOs ---

// Issued is what the display screen needs: the code plus its countdown.
type Issued struct {
	Code       string    `json:"code"`
	Kind       Kind      `json:"kind"`
	ExpiresAt  time.Time `json:"expires_at"`
	TTLSeconds int       `json:"ttl_seconds"`
}

func (t *Token) ToIssued(now time.Time) Issued {
	return Issued{
		Code:       t.Code,
		Kind:       t.Kind,
		ExpiresAt:  t.ExpiresAt,
		TTLSeconds: int(t.TTL(now).Seconds()),
	}
}

// ScannedStudent is what the teacher's screen shows right after a scan: enough
// context to pick an amount and confirm (flow A step 3).
type ScannedStudent struct {
	EnrollmentID kernel.EnrollmentID `json:"enrollment_id"`
	ClassroomID  kernel.ClassroomID  `json:"classroom_id"`
	StudentName  string              `json:"student_name"`
	StudentEmail string              `json:"student_email"`
	Balance      int64               `json:"balance"`
	TeamID       *kernel.TeamID      `json:"team_id"`
	TeamName     *string             `json:"team_name"`
	// GrantKey is the idempotency key the scanner must send with the grant it
	// is about to confirm. It is derived from the scanned token, so a double
	// tap on "confirm" — or a retry over a flaky network — pays only once
	// (§11.3). A deliberate second grant needs a fresh scan.
	GrantKey string `json:"grant_key"`
}
