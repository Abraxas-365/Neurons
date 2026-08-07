// Package userlookup adapts the IAM user repository to the narrow lookup
// interfaces the gamification services depend on, so those services never
// import IAM internals directly.
package userlookup

import (
	"context"
	"net/http"
	"strings"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/iam/user"
	"github.com/Abraxas-356/neurons/internal/kernel"
)

var ErrRegistry = errx.NewRegistry("USER_LOOKUP")

var CodeUserNotFound = ErrRegistry.Register(
	"NOT_FOUND",
	errx.TypeNotFound,
	http.StatusNotFound,
	"No user found with that email in this organization",
)

func ErrUserNotFound() error { return ErrRegistry.New(CodeUserNotFound) }

// Profile is the minimal user data the gamification modules display.
type Profile struct {
	ID    kernel.UserID `json:"id" db:"id"`
	Name  string        `json:"name" db:"name"`
	Email string        `json:"email" db:"email"`
}

type Lookup struct {
	repo user.UserRepository
}

func New(repo user.UserRepository) *Lookup {
	return &Lookup{repo: repo}
}

// FindIDByEmail resolves an email to a user id inside a tenant.
func (l *Lookup) FindIDByEmail(ctx context.Context, tenantID kernel.TenantID, email string) (kernel.UserID, error) {
	u, err := l.repo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)), tenantID)
	if err != nil {
		return "", ErrUserNotFound()
	}
	if u == nil {
		return "", ErrUserNotFound()
	}
	return u.ID, nil
}
