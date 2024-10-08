package user

import "time"

type User struct {
	ID         int64     `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Email      string    `json:"email" db:"email"`
	Role       string    `json:"role" db:"role"`
	AuthUserID string    `json:"auth_user_id" db:"auth_user_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
