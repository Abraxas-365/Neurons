package medalinfra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/gamification/dbx"
	"github.com/Abraxas-356/neurons/internal/gamification/medal"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/jmoiron/sqlx"
)

type PostgresMedalRepository struct {
	db *sqlx.DB
}

func NewPostgresMedalRepository(db *sqlx.DB) medal.Repository {
	return &PostgresMedalRepository{db: db}
}

const medalColumns = `id, classroom_id, name, description, image_url, icon, category, type, condition,
	max_awards, repeatable, show_on_member_profile, visible_to_students, available_from,
	available_until, is_active, created_at, updated_at`

const awardColumns = `id, medal_id, classroom_id, enrollment_id, team_id, awarded_by, note, awarded_at, revoked_at`

func (r *PostgresMedalRepository) Create(ctx context.Context, e *medal.Medal) error {
	query := `INSERT INTO medals (` + medalColumns + `)
	          VALUES (:id, :classroom_id, :name, :description, :image_url, :icon, :category, :type, :condition,
	                  :max_awards, :repeatable, :show_on_member_profile, :visible_to_students, :available_from,
	                  :available_until, :is_active, :created_at, :updated_at)`
	if _, err := r.db.NamedExecContext(ctx, query, e); err != nil {
		return errx.Wrap(err, "create medal", errx.TypeInternal)
	}
	return nil
}

func (r *PostgresMedalRepository) Update(ctx context.Context, e *medal.Medal) error {
	query := `UPDATE medals SET name = :name, description = :description, image_url = :image_url, icon = :icon,
	                 category = :category, type = :type, condition = :condition, max_awards = :max_awards,
	                 repeatable = :repeatable, show_on_member_profile = :show_on_member_profile,
	                 visible_to_students = :visible_to_students, available_from = :available_from,
	                 available_until = :available_until, is_active = :is_active
	          WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return errx.Wrap(err, "update medal", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return medal.ErrNotFound()
	}
	return nil
}

func (r *PostgresMedalRepository) GetByID(ctx context.Context, id kernel.MedalID) (*medal.Medal, error) {
	var e medal.Medal
	if err := r.db.GetContext(ctx, &e, `SELECT `+medalColumns+` FROM medals WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, medal.ErrNotFound()
		}
		return nil, errx.Wrap(err, "get medal", errx.TypeInternal)
	}
	return &e, nil
}

func (r *PostgresMedalRepository) ListByClassroom(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	activeOnly bool,
) ([]medal.Medal, error) {
	items := []medal.Medal{}
	query := `SELECT ` + medalColumns + ` FROM medals WHERE classroom_id = $1`
	if activeOnly {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY category NULLS LAST, name`
	if err := r.db.SelectContext(ctx, &items, query, classroomID); err != nil {
		return nil, errx.Wrap(err, "list medals", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresMedalRepository) Delete(ctx context.Context, id kernel.MedalID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM medals WHERE id = $1`, id)
	if err != nil {
		return errx.Wrap(err, "delete medal", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return medal.ErrNotFound()
	}
	return nil
}

func (r *PostgresMedalRepository) AwardMany(ctx context.Context, awards []medal.Award) error {
	if len(awards) == 0 {
		return nil
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errx.Wrap(err, "begin award tx", errx.TypeInternal)
	}
	defer tx.Rollback()

	query := `INSERT INTO medal_awards (` + awardColumns + `)
	          VALUES (:id, :medal_id, :classroom_id, :enrollment_id, :team_id, :awarded_by, :note, :awarded_at, :revoked_at)`
	for i := range awards {
		if _, err := tx.NamedExecContext(ctx, query, awards[i]); err != nil {
			return errx.Wrap(err, "insert medal award", errx.TypeInternal)
		}
	}

	if err := tx.Commit(); err != nil {
		return errx.Wrap(err, "commit award tx", errx.TypeInternal)
	}
	return nil
}

func (r *PostgresMedalRepository) GetAward(ctx context.Context, awardID string) (*medal.Award, error) {
	var a medal.Award
	query := `SELECT ` + dbx.Prefixed("a", awardColumns) + `,
	                 m.name AS medal_name, m.icon AS medal_icon, m.image_url AS medal_image_url
	          FROM medal_awards a
	          JOIN medals m ON m.id = a.medal_id
	          WHERE a.id = $1`
	if err := r.db.GetContext(ctx, &a, query, awardID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, medal.ErrAwardNotFound()
		}
		return nil, errx.Wrap(err, "get medal award", errx.TypeInternal)
	}
	return &a, nil
}

func (r *PostgresMedalRepository) RevokeAward(ctx context.Context, awardID string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE medal_awards SET revoked_at = CURRENT_TIMESTAMP WHERE id = $1 AND revoked_at IS NULL`,
		awardID,
	)
	if err != nil {
		return errx.Wrap(err, "revoke medal award", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return medal.ErrAlreadyRevoked()
	}
	return nil
}

func (r *PostgresMedalRepository) CountAwards(ctx context.Context, id kernel.MedalID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM medal_awards WHERE medal_id = $1 AND revoked_at IS NULL`
	if err := r.db.GetContext(ctx, &count, query, id); err != nil {
		return 0, errx.Wrap(err, "count medal awards", errx.TypeInternal)
	}
	return count, nil
}

func (r *PostgresMedalRepository) HasAward(
	ctx context.Context,
	id kernel.MedalID,
	enrollmentID *kernel.EnrollmentID,
	teamID *kernel.TeamID,
) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (
	              SELECT 1 FROM medal_awards
	              WHERE medal_id = $1 AND revoked_at IS NULL
	                AND enrollment_id IS NOT DISTINCT FROM $2
	                AND team_id IS NOT DISTINCT FROM $3
	          )`
	if err := r.db.GetContext(ctx, &exists, query, id, enrollmentID, teamID); err != nil {
		return false, errx.Wrap(err, "check medal award", errx.TypeInternal)
	}
	return exists, nil
}

func (r *PostgresMedalRepository) ListAwardsByClassroom(
	ctx context.Context,
	classroomID kernel.ClassroomID,
) ([]medal.Award, error) {
	items := []medal.Award{}
	query := `SELECT ` + dbx.Prefixed("a", awardColumns) + `,
	                 m.name AS medal_name, m.icon AS medal_icon, m.image_url AS medal_image_url,
	                 u.name AS student_name, t.name AS team_name
	          FROM medal_awards a
	          JOIN medals m ON m.id = a.medal_id
	          LEFT JOIN enrollments e ON e.id = a.enrollment_id
	          LEFT JOIN users u ON u.id = e.user_id
	          LEFT JOIN teams t ON t.id = a.team_id
	          WHERE a.classroom_id = $1
	          ORDER BY a.awarded_at DESC`
	if err := r.db.SelectContext(ctx, &items, query, classroomID); err != nil {
		return nil, errx.Wrap(err, "list medal awards", errx.TypeInternal)
	}
	return items, nil
}

// ListAwardsForStudent returns individual awards plus team awards the student
// inherits through team membership (RN-14).
func (r *PostgresMedalRepository) ListAwardsForStudent(
	ctx context.Context,
	enrollmentID kernel.EnrollmentID,
) ([]medal.Award, error) {
	items := []medal.Award{}
	query := `SELECT ` + dbx.Prefixed("a", awardColumns) + `,
	                 m.name AS medal_name, m.icon AS medal_icon, m.image_url AS medal_image_url,
	                 t.name AS team_name
	          FROM medal_awards a
	          JOIN medals m ON m.id = a.medal_id
	          LEFT JOIN teams t ON t.id = a.team_id
	          WHERE a.revoked_at IS NULL
	            AND (
	                a.enrollment_id = $1
	                OR (
	                    m.show_on_member_profile = TRUE
	                    AND a.team_id IN (SELECT team_id FROM enrollments WHERE id = $1 AND team_id IS NOT NULL)
	                )
	            )
	          ORDER BY a.awarded_at DESC`
	if err := r.db.SelectContext(ctx, &items, query, enrollmentID); err != nil {
		return nil, errx.Wrap(err, "list student medals", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresMedalRepository) ListAwardsByTeam(
	ctx context.Context,
	teamID kernel.TeamID,
) ([]medal.Award, error) {
	items := []medal.Award{}
	query := `SELECT ` + dbx.Prefixed("a", awardColumns) + `,
	                 m.name AS medal_name, m.icon AS medal_icon, m.image_url AS medal_image_url
	          FROM medal_awards a
	          JOIN medals m ON m.id = a.medal_id
	          WHERE a.team_id = $1 AND a.revoked_at IS NULL
	          ORDER BY a.awarded_at DESC`
	if err := r.db.SelectContext(ctx, &items, query, teamID); err != nil {
		return nil, errx.Wrap(err, "list team medals", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresMedalRepository) CountByEnrollment(
	ctx context.Context,
	enrollmentID kernel.EnrollmentID,
) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM medal_awards WHERE enrollment_id = $1 AND revoked_at IS NULL`
	if err := r.db.GetContext(ctx, &count, query, enrollmentID); err != nil {
		return 0, errx.Wrap(err, "count student medals", errx.TypeInternal)
	}
	return count, nil
}
