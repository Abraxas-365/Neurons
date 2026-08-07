package reasoninfra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/gamification/reason"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/jmoiron/sqlx"
)

type PostgresReasonRepository struct {
	db *sqlx.DB
}

func NewPostgresReasonRepository(db *sqlx.DB) reason.Repository {
	return &PostgresReasonRepository{db: db}
}

const reasonColumns = `id, classroom_id, name, description, icon, suggested_amount, scope, is_active, created_at, updated_at`

func (r *PostgresReasonRepository) Create(ctx context.Context, e *reason.Reason) error {
	query := `INSERT INTO reasons (` + reasonColumns + `)
	          VALUES (:id, :classroom_id, :name, :description, :icon, :suggested_amount, :scope, :is_active, :created_at, :updated_at)`
	if _, err := r.db.NamedExecContext(ctx, query, e); err != nil {
		return errx.Wrap(err, "create reason", errx.TypeInternal)
	}
	return nil
}

func (r *PostgresReasonRepository) Update(ctx context.Context, e *reason.Reason) error {
	query := `UPDATE reasons SET name = :name, description = :description, icon = :icon,
	                 suggested_amount = :suggested_amount, scope = :scope, is_active = :is_active
	          WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return errx.Wrap(err, "update reason", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return reason.ErrNotFound()
	}
	return nil
}

func (r *PostgresReasonRepository) GetByID(ctx context.Context, id kernel.ReasonID) (*reason.Reason, error) {
	var e reason.Reason
	if err := r.db.GetContext(ctx, &e, `SELECT `+reasonColumns+` FROM reasons WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, reason.ErrNotFound()
		}
		return nil, errx.Wrap(err, "get reason", errx.TypeInternal)
	}
	return &e, nil
}

func (r *PostgresReasonRepository) ListByClassroom(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	activeOnly bool,
) ([]reason.Reason, error) {
	items := []reason.Reason{}
	query := `SELECT ` + reasonColumns + ` FROM reasons WHERE classroom_id = $1`
	if activeOnly {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY name`
	if err := r.db.SelectContext(ctx, &items, query, classroomID); err != nil {
		return nil, errx.Wrap(err, "list reasons", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresReasonRepository) Delete(ctx context.Context, id kernel.ReasonID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM reasons WHERE id = $1`, id)
	if err != nil {
		return errx.Wrap(err, "delete reason", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return reason.ErrNotFound()
	}
	return nil
}
