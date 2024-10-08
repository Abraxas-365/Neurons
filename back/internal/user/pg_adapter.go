package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (name, email, role, created_at)
		VALUES (:name, :email, :role, :created_at)
		RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, user)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to create user: %v", err))
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&user.ID)
		if err != nil {
			return errors.ErrDatabase(fmt.Sprintf("failed to scan user ID: %v", err))
		}
	}
	return nil
}

func (r *PostgresRepository) GetUser(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT id, name, email, role, created_at
		FROM users
		WHERE id = $1
	`
	var user User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound("user not found")
		}
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to get user: %v", err))
	}
	return &user, nil
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, name, email, role, created_at
		FROM users
		WHERE email = $1
	`
	var user User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound("user not found")
		}
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to get user by email: %v", err))
	}
	return &user, nil
}

func (r *PostgresRepository) UpdateUser(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET name = :name, email = :email, role = :role
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to update user: %v", err))
	}
	return nil
}

func (r *PostgresRepository) DeleteUser(ctx context.Context, id int64) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to delete user: %v", err))
	}
	return nil
}

func (r *PostgresRepository) ListUsers(ctx context.Context, limit, offset int) ([]*User, error) {
	query := `
		SELECT id, name, email, role, created_at
		FROM users
		ORDER BY id
		LIMIT $1 OFFSET $2
	`
	var users []*User
	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to list users: %v", err))
	}
	return users, nil
}

func (r *PostgresRepository) ListUsersByRole(ctx context.Context, role string, limit, offset int) ([]*User, error) {
	query := `
		SELECT id, name, email, role, created_at
		FROM users
		WHERE role = $1
		ORDER BY id
		LIMIT $2 OFFSET $3
	`
	var users []*User
	err := r.db.SelectContext(ctx, &users, query, role, limit, offset)
	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to list users by role: %v", err))
	}
	return users, nil
}

func (r *PostgresRepository) ChangeUserRole(ctx context.Context, userID int64, newRole string) error {
	query := `
		UPDATE users
		SET role = $1
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, newRole, userID)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to change user role: %v", err))
	}
	return nil
}

func (r *PostgresRepository) CountUsers(ctx context.Context) (int64, error) {
	query := "SELECT COUNT(*) FROM users"
	var count int64
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, errors.ErrDatabase(fmt.Sprintf("failed to count users: %v", err))
	}
	return count, nil
}

func (r *PostgresRepository) CountUsersByRole(ctx context.Context, role string) (int64, error) {
	query := "SELECT COUNT(*) FROM users WHERE role = $1"
	var count int64
	err := r.db.GetContext(ctx, &count, query, role)
	if err != nil {
		return 0, errors.ErrDatabase(fmt.Sprintf("failed to count users by role: %v", err))
	}
	return count, nil
}

func (r *PostgresRepository) GetUserByAuthUserID(ctx context.Context, authUserID string) (*User, error) {
	query := `
		SELECT id, auth_user_id, name, email, role, created_at
		FROM users
		WHERE auth_user_id = $1
	`
	var user User
	err := r.db.GetContext(ctx, &user, query, authUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound("user not found")
		}
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to get user by auth_user_id: %v", err))
	}
	return &user, nil
}
