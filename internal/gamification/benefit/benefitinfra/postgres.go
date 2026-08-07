package benefitinfra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/gamification/benefit"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/jmoiron/sqlx"
)

type PostgresBenefitRepository struct {
	db *sqlx.DB
}

func NewPostgresBenefitRepository(db *sqlx.DB) benefit.Repository {
	return &PostgresBenefitRepository{db: db}
}

const benefitColumns = `id, classroom_id, name, description, icon, cost, max_uses, uses_count,
	max_uses_per_student, requires_approval, scope, conditions, available_from, available_until,
	is_active, created_at, updated_at`

func (r *PostgresBenefitRepository) Create(ctx context.Context, e *benefit.Benefit) error {
	query := `INSERT INTO benefits (` + benefitColumns + `)
	          VALUES (:id, :classroom_id, :name, :description, :icon, :cost, :max_uses, :uses_count,
	                  :max_uses_per_student, :requires_approval, :scope, :conditions, :available_from,
	                  :available_until, :is_active, :created_at, :updated_at)`
	if _, err := r.db.NamedExecContext(ctx, query, e); err != nil {
		return errx.Wrap(err, "create benefit", errx.TypeInternal)
	}
	return nil
}

func (r *PostgresBenefitRepository) Update(ctx context.Context, e *benefit.Benefit) error {
	query := `UPDATE benefits SET name = :name, description = :description, icon = :icon, cost = :cost,
	                 max_uses = :max_uses, max_uses_per_student = :max_uses_per_student,
	                 requires_approval = :requires_approval, scope = :scope, conditions = :conditions,
	                 available_from = :available_from, available_until = :available_until, is_active = :is_active
	          WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return errx.Wrap(err, "update benefit", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return benefit.ErrNotFound()
	}
	return nil
}

func (r *PostgresBenefitRepository) GetByID(ctx context.Context, id kernel.BenefitID) (*benefit.Benefit, error) {
	var e benefit.Benefit
	if err := r.db.GetContext(ctx, &e, `SELECT `+benefitColumns+` FROM benefits WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, benefit.ErrNotFound()
		}
		return nil, errx.Wrap(err, "get benefit", errx.TypeInternal)
	}
	return &e, nil
}

func (r *PostgresBenefitRepository) ListByClassroom(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	activeOnly bool,
) ([]benefit.Benefit, error) {
	items := []benefit.Benefit{}
	query := `SELECT ` + benefitColumns + ` FROM benefits WHERE classroom_id = $1`
	if activeOnly {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY name`
	if err := r.db.SelectContext(ctx, &items, query, classroomID); err != nil {
		return nil, errx.Wrap(err, "list benefits", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresBenefitRepository) Delete(ctx context.Context, id kernel.BenefitID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM benefits WHERE id = $1`, id)
	if err != nil {
		return errx.Wrap(err, "delete benefit", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return benefit.ErrNotFound()
	}
	return nil
}

// CountUsesByStudent counts applied redemptions of this benefit by one student.
// Reversed transactions do not count against the quota.
func (r *PostgresBenefitRepository) CountUsesByStudent(
	ctx context.Context,
	id kernel.BenefitID,
	enrollmentID kernel.EnrollmentID,
) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM transactions
	          WHERE benefit_id = $1 AND enrollment_id = $2
	            AND type = 'REDEMPTION' AND status = 'APPLIED'`
	if err := r.db.GetContext(ctx, &count, query, id, enrollmentID); err != nil {
		return 0, errx.Wrap(err, "count benefit uses", errx.TypeInternal)
	}
	return count, nil
}
