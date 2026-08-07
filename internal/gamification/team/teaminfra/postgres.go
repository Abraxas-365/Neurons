package teaminfra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/gamification/dbx"
	"github.com/Abraxas-356/neurons/internal/gamification/team"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresTeamRepository struct {
	db *sqlx.DB
}

func NewPostgresTeamRepository(db *sqlx.DB) team.Repository {
	return &PostgresTeamRepository{db: db}
}

const teamColumns = `id, classroom_id, name, description, color, icon, status, created_at, updated_at`

func (r *PostgresTeamRepository) Create(ctx context.Context, t *team.Team) error {
	query := `INSERT INTO teams (` + teamColumns + `)
	          VALUES (:id, :classroom_id, :name, :description, :color, :icon, :status, :created_at, :updated_at)`
	if _, err := r.db.NamedExecContext(ctx, query, t); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return team.ErrAlreadyExists()
		}
		return errx.Wrap(err, "create team", errx.TypeInternal)
	}
	return nil
}

func (r *PostgresTeamRepository) Update(ctx context.Context, t *team.Team) error {
	query := `UPDATE teams SET name = :name, description = :description, color = :color,
	                 icon = :icon, status = :status
	          WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, t)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return team.ErrAlreadyExists()
		}
		return errx.Wrap(err, "update team", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return team.ErrNotFound()
	}
	return nil
}

func (r *PostgresTeamRepository) GetByID(ctx context.Context, id kernel.TeamID) (*team.Team, error) {
	var t team.Team
	query := `SELECT ` + dbx.Prefixed("t", teamColumns) + `,
			(SELECT COUNT(*) FROM enrollments e WHERE e.team_id = t.id AND e.status = 'ACTIVE') AS member_count
		FROM teams t WHERE t.id = $1`
	if err := r.db.GetContext(ctx, &t, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, team.ErrNotFound()
		}
		return nil, errx.Wrap(err, "get team", errx.TypeInternal)
	}
	return &t, nil
}

func (r *PostgresTeamRepository) ListByClassroom(ctx context.Context, classroomID kernel.ClassroomID) ([]team.Team, error) {
	items := []team.Team{}
	query := `SELECT ` + dbx.Prefixed("t", teamColumns) + `,
			(SELECT COUNT(*) FROM enrollments e WHERE e.team_id = t.id AND e.status = 'ACTIVE') AS member_count
		FROM teams t WHERE t.classroom_id = $1 ORDER BY t.name`
	if err := r.db.SelectContext(ctx, &items, query, classroomID); err != nil {
		return nil, errx.Wrap(err, "list teams", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresTeamRepository) Delete(ctx context.Context, id kernel.TeamID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM teams WHERE id = $1`, id)
	if err != nil {
		return errx.Wrap(err, "delete team", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return team.ErrNotFound()
	}
	return nil
}

func (r *PostgresTeamRepository) Members(ctx context.Context, id kernel.TeamID) ([]team.Member, error) {
	items := []team.Member{}
	query := `SELECT e.id AS enrollment_id, e.user_id, u.name, u.email, e.balance,
			COALESCE(tm.is_coordinator, FALSE) AS is_coordinator,
			e.joined_at
		FROM enrollments e
		JOIN users u ON u.id = e.user_id
		LEFT JOIN team_memberships tm ON tm.enrollment_id = e.id AND tm.team_id = e.team_id AND tm.left_at IS NULL
		WHERE e.team_id = $1 AND e.status = 'ACTIVE'
		ORDER BY u.name`
	if err := r.db.SelectContext(ctx, &items, query, id); err != nil {
		return nil, errx.Wrap(err, "list team members", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresTeamRepository) SetCoordinator(ctx context.Context, id kernel.TeamID, enrollmentID kernel.EnrollmentID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errx.Wrap(err, "begin set coordinator", errx.TypeInternal)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE team_memberships SET is_coordinator = FALSE WHERE team_id = $1 AND left_at IS NULL`, id); err != nil {
		return errx.Wrap(err, "clear coordinators", errx.TypeInternal)
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE team_memberships SET is_coordinator = TRUE
		 WHERE team_id = $1 AND enrollment_id = $2 AND left_at IS NULL`, id, enrollmentID)
	if err != nil {
		return errx.Wrap(err, "set coordinator", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return team.ErrNotFound()
	}

	if err := tx.Commit(); err != nil {
		return errx.Wrap(err, "commit set coordinator", errx.TypeInternal)
	}
	return nil
}
