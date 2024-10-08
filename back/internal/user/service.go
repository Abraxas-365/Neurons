package user

import (
	"context"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"time"
)

// Servicer defines the interface for user-related operations
type Servicer interface {
	CreateUser(ctx context.Context, name, email, role string) (*User, error)
	GetUser(ctx context.Context, id int64) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id int64) error
	ListUsers(ctx context.Context, limit, offset int) ([]*User, error)
	ListUsersByRole(ctx context.Context, role string, limit, offset int) ([]*User, error)
	ChangeUserRole(ctx context.Context, userID int64, newRole string) error
	CountUsers(ctx context.Context) (int64, error)
	CountUsersByRole(ctx context.Context, role string) (int64, error)
	GetUserByAuthUserID(ctx context.Context, authUserID string) (*User, error)
}

// Ensure Service implements Servicer
var _ Servicer = (*Service)(nil)

// Service implements the Servicer interface
type Service struct {
	repo DBRepository
}

// NewService creates a new user service
func NewService(repo DBRepository) *Service {
	return &Service{repo: repo}
}

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, name, email, role string) (*User, error) {
	if name == "" || email == "" || role == "" {
		return nil, errors.ErrBadRequest("name, email, and role are required")
	}

	user := &User{
		Name:      name,
		Email:     email,
		Role:      role,
		CreatedAt: time.Now(),
	}

	err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, id int64) (*User, error) {
	user, err := s.repo.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser updates an existing user
func (s *Service) UpdateUser(ctx context.Context, user *User) error {
	if user.ID == 0 {
		return errors.ErrBadRequest("user ID is required")
	}
	err := s.repo.UpdateUser(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	err := s.repo.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

// ListUsers retrieves a list of users
func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]*User, error) {
	users, err := s.repo.ListUsers(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// ListUsersByRole retrieves a list of users with a specific role
func (s *Service) ListUsersByRole(ctx context.Context, role string, limit, offset int) ([]*User, error) {
	users, err := s.repo.ListUsersByRole(ctx, role, limit, offset)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// ChangeUserRole changes the role of a user
func (s *Service) ChangeUserRole(ctx context.Context, userID int64, newRole string) error {
	if newRole == "" {
		return errors.ErrBadRequest("new role is required")
	}
	err := s.repo.ChangeUserRole(ctx, userID, newRole)
	if err != nil {
		return err
	}
	return nil
}

// CountUsers counts the total number of users
func (s *Service) CountUsers(ctx context.Context) (int64, error) {
	count, err := s.repo.CountUsers(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountUsersByRole counts the number of users with a specific role
func (s *Service) CountUsersByRole(ctx context.Context, role string) (int64, error) {
	count, err := s.repo.CountUsersByRole(ctx, role)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) GetUserByAuthUserID(ctx context.Context, authUserID string) (*User, error) {
	return s.repo.GetUserByAuthUserID(ctx, authUserID)
}
