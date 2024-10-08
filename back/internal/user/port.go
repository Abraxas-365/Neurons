package user

import (
	"context"
)

// DBRepository defines the interface for user database operations
type DBRepository interface {
	// CreateUser creates a new user in the database
	CreateUser(ctx context.Context, user *User) error

	// GetUser retrieves a user by their ID
	GetUser(ctx context.Context, id int64) (*User, error)

	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// UpdateUser updates an existing user in the database
	UpdateUser(ctx context.Context, user *User) error

	// DeleteUser deletes a user from the database
	DeleteUser(ctx context.Context, id int64) error

	// ListUsers retrieves a list of users, potentially with pagination
	ListUsers(ctx context.Context, limit, offset int) ([]*User, error)

	// ListUsersByRole retrieves a list of users with a specific role
	ListUsersByRole(ctx context.Context, role string, limit, offset int) ([]*User, error)

	// ChangeUserRole changes the role of a user
	ChangeUserRole(ctx context.Context, userID int64, newRole string) error

	// CountUsers counts the total number of users
	CountUsers(ctx context.Context) (int64, error)

	// CountUsersByRole counts the number of users with a specific role
	CountUsersByRole(ctx context.Context, role string) (int64, error)

	GetUserByAuthUserID(ctx context.Context, authUserID string) (*User, error)
}
