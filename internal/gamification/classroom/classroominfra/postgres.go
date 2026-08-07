package classroominfra

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/gamification/classroom"
	"github.com/Abraxas-356/neurons/internal/gamification/dbx"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresClassroomRepository struct {
	db *sqlx.DB
}

func NewPostgresClassroomRepository(db *sqlx.DB) classroom.Repository {
	return &PostgresClassroomRepository{db: db}
}

const classroomColumns = `id, tenant_id, name, section, term, description, icon, invite_code, status,
	unlimited_issuance, vault_balance, total_granted, total_redeemed, join_policy,
	void_window_seconds, reconfirm_threshold, allow_free_redemption, ranking_public,
	starts_at, ends_at, closed_at, created_by, created_at, updated_at`

func (r *PostgresClassroomRepository) Create(ctx context.Context, entity *classroom.Classroom) error {
	query := `INSERT INTO classrooms (` + classroomColumns + `)
	          VALUES (:id, :tenant_id, :name, :section, :term, :description, :icon, :invite_code, :status,
	                  :unlimited_issuance, :vault_balance, :total_granted, :total_redeemed, :join_policy,
	                  :void_window_seconds, :reconfirm_threshold, :allow_free_redemption, :ranking_public,
	                  :starts_at, :ends_at, :closed_at, :created_by, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return classroom.ErrClassroomAlreadyExists()
		}
		return errx.Wrap(err, "create classroom", errx.TypeInternal)
	}
	return nil
}

func (r *PostgresClassroomRepository) Update(ctx context.Context, entity *classroom.Classroom) error {
	query := `UPDATE classrooms SET
		name = :name, section = :section, term = :term, description = :description, icon = :icon,
		status = :status, unlimited_issuance = :unlimited_issuance, vault_balance = :vault_balance,
		total_granted = :total_granted, total_redeemed = :total_redeemed, join_policy = :join_policy,
		void_window_seconds = :void_window_seconds, reconfirm_threshold = :reconfirm_threshold,
		allow_free_redemption = :allow_free_redemption, ranking_public = :ranking_public,
		starts_at = :starts_at, ends_at = :ends_at, closed_at = :closed_at
		WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return errx.Wrap(err, "update classroom", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return classroom.ErrClassroomNotFound()
	}
	return nil
}

func (r *PostgresClassroomRepository) GetByID(ctx context.Context, id kernel.ClassroomID) (*classroom.Classroom, error) {
	var entity classroom.Classroom
	query := `SELECT ` + classroomColumns + ` FROM classrooms WHERE id = $1`
	if err := r.db.GetContext(ctx, &entity, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, classroom.ErrClassroomNotFound()
		}
		return nil, errx.Wrap(err, "get classroom", errx.TypeInternal)
	}
	return &entity, nil
}

func (r *PostgresClassroomRepository) GetByInviteCode(ctx context.Context, code string) (*classroom.Classroom, error) {
	var entity classroom.Classroom
	query := `SELECT ` + classroomColumns + ` FROM classrooms WHERE invite_code = $1`
	if err := r.db.GetContext(ctx, &entity, query, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, classroom.ErrInviteCodeInvalid()
		}
		return nil, errx.Wrap(err, "get classroom by invite code", errx.TypeInternal)
	}
	return &entity, nil
}

func (r *PostgresClassroomRepository) List(ctx context.Context, tenantID kernel.TenantID, opts kernel.PaginationOptions) (kernel.Paginated[classroom.Classroom], error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM classrooms WHERE tenant_id = $1`, tenantID); err != nil {
		return kernel.Paginated[classroom.Classroom]{}, errx.Wrap(err, "count classrooms", errx.TypeInternal)
	}

	offset := (opts.Page - 1) * opts.PageSize
	items := []classroom.Classroom{}
	if err := r.db.SelectContext(ctx, &items,
		`SELECT `+classroomColumns+` FROM classrooms WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PageSize, offset); err != nil {
		return kernel.Paginated[classroom.Classroom]{}, errx.Wrap(err, "list classrooms", errx.TypeInternal)
	}

	return kernel.NewPaginated(items, opts.Page, opts.PageSize, total), nil
}

func (r *PostgresClassroomRepository) ListForTeacher(ctx context.Context, tenantID kernel.TenantID, userID kernel.UserID, opts kernel.PaginationOptions) (kernel.Paginated[classroom.Classroom], error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM classrooms c
		JOIN classroom_teachers ct ON ct.classroom_id = c.id
		WHERE c.tenant_id = $1 AND ct.user_id = $2`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID, userID); err != nil {
		return kernel.Paginated[classroom.Classroom]{}, errx.Wrap(err, "count teacher classrooms", errx.TypeInternal)
	}

	offset := (opts.Page - 1) * opts.PageSize
	items := []classroom.Classroom{}
	query := `SELECT ` + dbx.Prefixed("c", classroomColumns) + ` FROM classrooms c
		JOIN classroom_teachers ct ON ct.classroom_id = c.id
		WHERE c.tenant_id = $1 AND ct.user_id = $2
		ORDER BY c.created_at DESC LIMIT $3 OFFSET $4`
	if err := r.db.SelectContext(ctx, &items, query, tenantID, userID, opts.PageSize, offset); err != nil {
		return kernel.Paginated[classroom.Classroom]{}, errx.Wrap(err, "list teacher classrooms", errx.TypeInternal)
	}

	return kernel.NewPaginated(items, opts.Page, opts.PageSize, total), nil
}

func (r *PostgresClassroomRepository) Delete(ctx context.Context, id kernel.ClassroomID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM classrooms WHERE id = $1`, id)
	if err != nil {
		return errx.Wrap(err, "delete classroom", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return classroom.ErrClassroomNotFound()
	}
	return nil
}

// --- Teachers ---

func (r *PostgresClassroomRepository) AddTeacher(ctx context.Context, t *classroom.ClassroomTeacher) error {
	query := `INSERT INTO classroom_teachers (classroom_id, user_id, role, grant_allowance, granted_from_allowance, added_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, t.ClassroomID, t.UserID, t.Role, t.GrantAllowance, t.GrantedFromAllowance, t.AddedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return classroom.ErrTeacherAlreadyAdded()
		}
		return errx.Wrap(err, "add classroom teacher", errx.TypeInternal)
	}
	return nil
}

func (r *PostgresClassroomRepository) RemoveTeacher(ctx context.Context, id kernel.ClassroomID, userID kernel.UserID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM classroom_teachers WHERE classroom_id = $1 AND user_id = $2 AND role <> 'OWNER'`, id, userID)
	if err != nil {
		return errx.Wrap(err, "remove classroom teacher", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return classroom.ErrCannotRemoveOwner()
	}
	return nil
}

func (r *PostgresClassroomRepository) GetTeacher(ctx context.Context, id kernel.ClassroomID, userID kernel.UserID) (*classroom.ClassroomTeacher, error) {
	var t classroom.ClassroomTeacher
	query := `SELECT ct.classroom_id, ct.user_id, ct.role, ct.grant_allowance, ct.granted_from_allowance,
	                 ct.added_at, u.name, u.email
	          FROM classroom_teachers ct
	          JOIN users u ON u.id = ct.user_id
	          WHERE ct.classroom_id = $1 AND ct.user_id = $2`
	if err := r.db.GetContext(ctx, &t, query, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, classroom.ErrNotTeacher()
		}
		return nil, errx.Wrap(err, "get classroom teacher", errx.TypeInternal)
	}
	return &t, nil
}

func (r *PostgresClassroomRepository) ListTeachers(ctx context.Context, id kernel.ClassroomID) ([]classroom.ClassroomTeacher, error) {
	items := []classroom.ClassroomTeacher{}
	query := `SELECT ct.classroom_id, ct.user_id, ct.role, ct.grant_allowance, ct.granted_from_allowance,
	                 ct.added_at, u.name, u.email
	          FROM classroom_teachers ct
	          JOIN users u ON u.id = ct.user_id
	          WHERE ct.classroom_id = $1
	          ORDER BY ct.role, ct.added_at`
	if err := r.db.SelectContext(ctx, &items, query, id); err != nil {
		return nil, errx.Wrap(err, "list classroom teachers", errx.TypeInternal)
	}
	return items, nil
}
