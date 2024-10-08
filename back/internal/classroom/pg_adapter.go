package classroom

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

func (r *PostgresRepository) CreateClassroom(ctx context.Context, classroom *Classroom) (*ClassroomWithData, error) {
	query := `
		INSERT INTO classrooms (name, teacher_id, available_neurons, created_at)
		VALUES (:name, :teacher_id, :available_neurons, :created_at)
		RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, classroom)
	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to create classroom: %v", err))
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&classroom.ID)
		if err != nil {
			return nil, errors.ErrDatabase(fmt.Sprintf("failed to scan classroom ID: %v", err))
		}
	}

	return r.GetClassroom(ctx, classroom.ID)
}

func (r *PostgresRepository) GetClassroom(ctx context.Context, id int64) (*ClassroomWithData, error) {
	query := `
		SELECT c.id, c.name, c.teacher_id, c.available_neurons, c.created_at,
			   u.id AS "teacher.id", u.name AS "teacher.name", u.email AS "teacher.email", 
			   u.role AS "teacher.role", u.created_at AS "teacher.created_at"
		FROM classrooms c
		JOIN users u ON c.teacher_id = u.id
		WHERE c.id = $1
	`
	var classroomWithData ClassroomWithData
	err := r.db.GetContext(ctx, &classroomWithData, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound("classroom not found")
		}
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to get classroom: %v", err))
	}

	students, err := r.GetClassroomStudents(ctx, id)
	if err != nil {
		return nil, err
	}
	classroomWithData.Students = students

	return &classroomWithData, nil
}

func (r *PostgresRepository) UpdateClassroom(ctx context.Context, classroom *Classroom) (*ClassroomWithData, error) {
	query := `
		UPDATE classrooms
		SET name = :name, teacher_id = :teacher_id, available_neurons = :available_neurons
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, classroom)
	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to update classroom: %v", err))
	}

	return r.GetClassroom(ctx, classroom.ID)
}

func (r *PostgresRepository) DeleteClassroom(ctx context.Context, id int64) error {
	query := "DELETE FROM classrooms WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to delete classroom: %v", err))
	}
	return nil
}

func (r *PostgresRepository) ListClassrooms(ctx context.Context, limit, offset int) ([]*ClassroomWithData, error) {
	query := `
		SELECT c.id, c.name, c.teacher_id, c.available_neurons, c.created_at,
			   u.id AS "teacher.id", u.name AS "teacher.name", u.email AS "teacher.email", 
			   u.role AS "teacher.role", u.created_at AS "teacher.created_at"
		FROM classrooms c
		JOIN users u ON c.teacher_id = u.id
		ORDER BY c.id
		LIMIT $1 OFFSET $2
	`
	var classroomsWithData []*ClassroomWithData
	err := r.db.SelectContext(ctx, &classroomsWithData, query, limit, offset)
	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to list classrooms: %v", err))
	}

	for _, classroomWithData := range classroomsWithData {
		students, err := r.GetClassroomStudents(ctx, classroomWithData.ID)
		if err != nil {
			return nil, err
		}
		classroomWithData.Students = students
	}

	return classroomsWithData, nil
}

func (r *PostgresRepository) AddStudentToClassroom(ctx context.Context, classroomID, studentID int64) error {
	query := `
		INSERT INTO users_classrooms (user_id, classroom_id)
		VALUES ($1, $2)
	`
	_, err := r.db.ExecContext(ctx, query, studentID, classroomID)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to add student to classroom: %v", err))
	}
	return nil
}

func (r *PostgresRepository) RemoveStudentFromClassroom(ctx context.Context, classroomID, studentID int64) error {
	query := `
		DELETE FROM users_classrooms
		WHERE user_id = $1 AND classroom_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, studentID, classroomID)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to remove student from classroom: %v", err))
	}
	return nil
}

func (r *PostgresRepository) UpdateAvailableNeurons(ctx context.Context, classroomID int64, neurons int) error {
	query := `
		UPDATE classrooms
		SET available_neurons = $1
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, neurons, classroomID)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to update available neurons: %v", err))
	}
	return nil
}

func (r *PostgresRepository) GetClassroomStudents(ctx context.Context, classroomID int64) ([]*Student, error) {
	query := `
		SELECT u.id, u.name, u.email, u.role, u.created_at, uc.neurons
		FROM users u
		JOIN users_classrooms uc ON u.id = uc.user_id
		WHERE uc.classroom_id = $1 AND u.role = 'student'
	`
	var students []*Student
	err := r.db.SelectContext(ctx, &students, query, classroomID)
	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to get classroom students: %v", err))
	}
	return students, nil
}

func (r *PostgresRepository) IsStudentInClassroom(ctx context.Context, classroomID, studentID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users_classrooms
			WHERE classroom_id = $1 AND user_id = $2
		)
	`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, classroomID, studentID)
	if err != nil {
		return false, errors.ErrDatabase(fmt.Sprintf("failed to check if student is in classroom: %v", err))
	}
	return exists, nil
}

func (r *PostgresRepository) TransferNeurons(ctx context.Context, classroomID, studentID int64, amount int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to begin transaction: %v", err))
	}
	defer tx.Rollback()

	// Decrease classroom neurons
	_, err = tx.ExecContext(ctx, `
		UPDATE classrooms
		SET available_neurons = available_neurons - $1
		WHERE id = $2
	`, amount, classroomID)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to decrease classroom neurons: %v", err))
	}

	// Increase student neurons
	_, err = tx.ExecContext(ctx, `
		UPDATE users_classrooms
		SET neurons = neurons + $1
		WHERE classroom_id = $2 AND user_id = $3
	`, amount, classroomID, studentID)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to increase student neurons: %v", err))
	}

	return tx.Commit()
}

func (r *PostgresRepository) RecordNeuronTransaction(ctx context.Context, transaction *NeuronTransaction) error {
	query := `
		INSERT INTO neuron_transactions (classroom_id, user_id, amount, transaction_type, created_at)
		VALUES (:classroom_id, :user_id, :amount, :transaction_type, :created_at)
		RETURNING id
	`
	rows, err := r.db.NamedQueryContext(ctx, query, transaction)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to record neuron transaction: %v", err))
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&transaction.ID)
		if err != nil {
			return errors.ErrDatabase(fmt.Sprintf("failed to scan transaction ID: %v", err))
		}
	}
	return nil
}

func (r *PostgresRepository) GetUserNeurons(ctx context.Context, userID, classroomID int64) (int, error) {
	query := `
		SELECT neurons
		FROM users_classrooms
		WHERE user_id = $1 AND classroom_id = $2
	`
	var neurons int
	err := r.db.GetContext(ctx, &neurons, query, userID, classroomID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.ErrNotFound("user not found in this classroom")
		}
		return 0, errors.ErrDatabase(fmt.Sprintf("failed to get user neurons: %v", err))
	}
	return neurons, nil
}

func (r *PostgresRepository) ListUserClassrooms(ctx context.Context, userID int64, role string, limit, offset int) ([]*ClassroomWithData, error) {
	var query string
	var args []interface{}

	if role == "teacher" {
		query = `
			SELECT c.id, c.name, c.teacher_id, c.available_neurons, c.created_at,
				   u.id AS "teacher.id", u.name AS "teacher.name", u.email AS "teacher.email", 
				   u.role AS "teacher.role", u.created_at AS "teacher.created_at"
			FROM classrooms c
			JOIN users u ON c.teacher_id = u.id
			WHERE c.teacher_id = $1
			ORDER BY c.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{userID, limit, offset}
	} else if role == "student" {
		query = `
			SELECT c.id, c.name, c.teacher_id, c.available_neurons, c.created_at,
				   u.id AS "teacher.id", u.name AS "teacher.name", u.email AS "teacher.email", 
				   u.role AS "teacher.role", u.created_at AS "teacher.created_at"
			FROM classrooms c
			JOIN users u ON c.teacher_id = u.id
			JOIN users_classrooms uc ON c.id = uc.classroom_id
			WHERE uc.user_id = $1
			ORDER BY c.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{userID, limit, offset}
	} else {
		return nil, errors.ErrBadRequest("invalid role")
	}

	var classroomsWithData []*ClassroomWithData
	err := r.db.SelectContext(ctx, &classroomsWithData, query, args...)
	if err != nil {
		return nil, errors.ErrDatabase(fmt.Sprintf("failed to list user classrooms: %v", err))
	}

	for _, classroomWithData := range classroomsWithData {
		students, err := r.GetClassroomStudents(ctx, classroomWithData.ID)
		if err != nil {
			return nil, err
		}
		classroomWithData.Students = students
	}

	return classroomsWithData, nil
}

func (r *PostgresRepository) TransferNeuronsToClassroom(ctx context.Context, classroomID, studentID int64, amount int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to begin transaction: %v", err))
	}
	defer tx.Rollback()

	// Increase classroom neurons
	_, err = tx.ExecContext(ctx, `
        UPDATE classrooms
        SET available_neurons = available_neurons + $1
        WHERE id = $2
    `, amount, classroomID)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to increase classroom neurons: %v", err))
	}

	// Decrease student neurons
	_, err = tx.ExecContext(ctx, `
        UPDATE users_classrooms
        SET neurons = neurons - $1
        WHERE classroom_id = $2 AND user_id = $3
    `, amount, classroomID, studentID)
	if err != nil {
		return errors.ErrDatabase(fmt.Sprintf("failed to decrease student neurons: %v", err))
	}

	return tx.Commit()
}
