package classroom

import (
	"time"

	"github.com/Abraxas-365/neurons/internal/user"
)

type Student struct {
	user.User `json:"user"`
	Neurons   int `json:"neurons"`
}

// Classroom represents a classroom in the education gamification system
type Classroom struct {
	ID               int64     `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	TeacherID        int64     `json:"teacher_id" db:"teacher_id"`
	AvailableNeurons int       `json:"available_neurons" db:"available_neurons"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// ClassroomWithData represents a classroom with additional data about the teacher and students
type ClassroomWithData struct {
	Classroom
	Teacher  user.User  `json:"teacher"`
	Students []*Student `json:"students"`
}

// UserClassroom represents the relationship between a user (student) and a classroom
type UserClassroom struct {
	UserID      int64 `json:"user_id" db:"user_id"`
	ClassroomID int64 `json:"classroom_id" db:"classroom_id"`
	Neurons     int   `json:"neurons" db:"neurons"`
}

// NeuronTransaction represents a transaction of neurons
type NeuronTransaction struct {
	ID              int64     `json:"id" db:"id"`
	ClassroomID     int64     `json:"classroom_id" db:"classroom_id"`
	UserID          int64     `json:"user_id" db:"user_id"`
	Amount          int       `json:"amount" db:"amount"`
	TransactionType string    `json:"transaction_type" db:"transaction_type"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}
