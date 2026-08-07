package enrollmentinfra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/gamification/dbx"
	"github.com/Abraxas-356/neurons/internal/gamification/enrollment"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresEnrollmentRepository struct {
	db *sqlx.DB
}

func NewPostgresEnrollmentRepository(db *sqlx.DB) enrollment.Repository {
	return &PostgresEnrollmentRepository{db: db}
}

const enrollmentColumns = `id, classroom_id, user_id, balance, total_received, total_returned,
	team_id, status, student_code, joined_at, left_at, last_activity_at, created_at, updated_at`

func (r *PostgresEnrollmentRepository) Create(ctx context.Context, e *enrollment.Enrollment) error {
	query := `INSERT INTO enrollments (` + enrollmentColumns + `)
	          VALUES (:id, :classroom_id, :user_id, :balance, :total_received, :total_returned,
	                  :team_id, :status, :student_code, :joined_at, :left_at, :last_activity_at,
	                  :created_at, :updated_at)`
	if _, err := r.db.NamedExecContext(ctx, query, e); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return enrollment.ErrAlreadyEnrolled()
		}
		return errx.Wrap(err, "create enrollment", errx.TypeInternal)
	}
	return nil
}

func (r *PostgresEnrollmentRepository) Update(ctx context.Context, e *enrollment.Enrollment) error {
	query := `UPDATE enrollments SET
		status = :status, student_code = :student_code, team_id = :team_id,
		left_at = :left_at, last_activity_at = :last_activity_at
		WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return errx.Wrap(err, "update enrollment", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return enrollment.ErrNotFound()
	}
	return nil
}

func (r *PostgresEnrollmentRepository) GetByID(ctx context.Context, id kernel.EnrollmentID) (*enrollment.Enrollment, error) {
	var e enrollment.Enrollment
	query := `SELECT ` + dbx.Prefixed("e", enrollmentColumns) + `, u.name, u.email, t.name AS team_name
	          FROM enrollments e
	          JOIN users u ON u.id = e.user_id
	          LEFT JOIN teams t ON t.id = e.team_id
	          WHERE e.id = $1`
	if err := r.db.GetContext(ctx, &e, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, enrollment.ErrNotFound()
		}
		return nil, errx.Wrap(err, "get enrollment", errx.TypeInternal)
	}
	return &e, nil
}

func (r *PostgresEnrollmentRepository) GetByUserAndClassroom(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	userID kernel.UserID,
) (*enrollment.Enrollment, error) {
	var e enrollment.Enrollment
	query := `SELECT ` + dbx.Prefixed("e", enrollmentColumns) + `, u.name, u.email, t.name AS team_name
	          FROM enrollments e
	          JOIN users u ON u.id = e.user_id
	          LEFT JOIN teams t ON t.id = e.team_id
	          WHERE e.classroom_id = $1 AND e.user_id = $2`
	if err := r.db.GetContext(ctx, &e, query, classroomID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, enrollment.ErrNotFound()
		}
		return nil, errx.Wrap(err, "get enrollment by user", errx.TypeInternal)
	}
	return &e, nil
}

// rosterSortColumns whitelists sortable columns so the sort parameter can never
// be used for SQL injection.
var rosterSortColumns = map[string]string{
	"name":     "u.name",
	"balance":  "e.balance",
	"received": "e.total_received",
	"returned": "e.total_returned",
	"activity": "e.last_activity_at",
	"joined":   "e.joined_at",
}

func (r *PostgresEnrollmentRepository) Roster(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	filter enrollment.RosterFilter,
	opts kernel.PaginationOptions,
) (kernel.Paginated[enrollment.Enrollment], error) {
	var empty kernel.Paginated[enrollment.Enrollment]

	where := []string{"e.classroom_id = ?"}
	args := []any{classroomID}

	if filter.Status != nil {
		where = append(where, "e.status = ?")
		args = append(args, *filter.Status)
	}
	if filter.TeamID != nil {
		where = append(where, "e.team_id = ?")
		args = append(args, *filter.TeamID)
	}
	if s := strings.TrimSpace(filter.Search); s != "" {
		where = append(where, "(u.name ILIKE ? OR u.email ILIKE ? OR e.student_code ILIKE ?)")
		like := "%" + s + "%"
		args = append(args, like, like, like)
	}

	whereClause := strings.Join(where, " AND ")

	countQuery := `SELECT COUNT(*) FROM enrollments e JOIN users u ON u.id = e.user_id WHERE ` + whereClause
	var total int
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(countQuery), args...); err != nil {
		return empty, errx.Wrap(err, "count roster", errx.TypeInternal)
	}

	orderCol, ok := rosterSortColumns[filter.SortBy]
	if !ok {
		orderCol = "u.name"
	}
	dir := "ASC"
	if strings.EqualFold(filter.SortDir, "desc") {
		dir = "DESC"
	}

	offset := (opts.Page - 1) * opts.PageSize
	listQuery := fmt.Sprintf(`SELECT %s, u.name, u.email, t.name AS team_name,
			(SELECT COUNT(*) FROM medal_awards ma WHERE ma.enrollment_id = e.id AND ma.revoked_at IS NULL) AS medal_count
		FROM enrollments e
		JOIN users u ON u.id = e.user_id
		LEFT JOIN teams t ON t.id = e.team_id
		WHERE %s
		ORDER BY %s %s NULLS LAST
		LIMIT ? OFFSET ?`, dbx.Prefixed("e", enrollmentColumns), whereClause, orderCol, dir)

	items := []enrollment.Enrollment{}
	listArgs := append(append([]any{}, args...), opts.PageSize, offset)
	if err := r.db.SelectContext(ctx, &items, r.db.Rebind(listQuery), listArgs...); err != nil {
		return empty, errx.Wrap(err, "list roster", errx.TypeInternal)
	}

	return kernel.NewPaginated(items, opts.Page, opts.PageSize, total), nil
}

func (r *PostgresEnrollmentRepository) ListByUser(
	ctx context.Context,
	tenantID kernel.TenantID,
	userID kernel.UserID,
) ([]enrollment.MyEnrollmentRow, error) {
	rows := []enrollment.MyEnrollmentRow{}
	query := `SELECT ` + dbx.Prefixed("e", enrollmentColumns) + `, u.name, u.email, t.name AS team_name,
			c.name AS classroom_name, c.section, c.term, c.icon, c.status AS classroom_status
		FROM enrollments e
		JOIN classrooms c ON c.id = e.classroom_id
		JOIN users u ON u.id = e.user_id
		LEFT JOIN teams t ON t.id = e.team_id
		WHERE e.user_id = $1 AND c.tenant_id = $2 AND e.status <> 'WITHDRAWN'
		ORDER BY c.created_at DESC`
	if err := r.db.SelectContext(ctx, &rows, query, userID, tenantID); err != nil {
		return nil, errx.Wrap(err, "list enrollments by user", errx.TypeInternal)
	}
	return rows, nil
}

func (r *PostgresEnrollmentRepository) ListActiveByTeam(ctx context.Context, teamID kernel.TeamID) ([]enrollment.Enrollment, error) {
	items := []enrollment.Enrollment{}
	query := `SELECT ` + dbx.Prefixed("e", enrollmentColumns) + `, u.name, u.email
		FROM enrollments e
		JOIN users u ON u.id = e.user_id
		WHERE e.team_id = $1 AND e.status = 'ACTIVE'
		ORDER BY u.name`
	if err := r.db.SelectContext(ctx, &items, query, teamID); err != nil {
		return nil, errx.Wrap(err, "list team members", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresEnrollmentRepository) ListActiveByIDs(
	ctx context.Context,
	classroomID kernel.ClassroomID,
	ids []kernel.EnrollmentID,
) ([]enrollment.Enrollment, error) {
	if len(ids) == 0 {
		return []enrollment.Enrollment{}, nil
	}

	raw := make([]string, 0, len(ids))
	for _, id := range ids {
		raw = append(raw, id.String())
	}

	items := []enrollment.Enrollment{}
	query := `SELECT ` + dbx.Prefixed("e", enrollmentColumns) + `, u.name, u.email
		FROM enrollments e
		JOIN users u ON u.id = e.user_id
		WHERE e.classroom_id = $1 AND e.status = 'ACTIVE' AND e.id = ANY($2)
		ORDER BY u.name`
	if err := r.db.SelectContext(ctx, &items, query, classroomID, pq.Array(raw)); err != nil {
		return nil, errx.Wrap(err, "list enrollments by ids", errx.TypeInternal)
	}
	return items, nil
}

func (r *PostgresEnrollmentRepository) CountActive(ctx context.Context, classroomID kernel.ClassroomID) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM enrollments WHERE classroom_id = $1 AND status = 'ACTIVE'`, classroomID); err != nil {
		return 0, errx.Wrap(err, "count active enrollments", errx.TypeInternal)
	}
	return count, nil
}

// SetTeam moves a student between teams, closing the previous membership row
// and opening a new one so the history is preserved.
func (r *PostgresEnrollmentRepository) SetTeam(ctx context.Context, id kernel.EnrollmentID, teamID *kernel.TeamID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errx.Wrap(err, "begin set team", errx.TypeInternal)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE team_memberships SET left_at = NOW() WHERE enrollment_id = $1 AND left_at IS NULL`, id); err != nil {
		return errx.Wrap(err, "close team membership", errx.TypeInternal)
	}

	if teamID != nil {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO team_memberships (id, team_id, enrollment_id, joined_at)
			 VALUES (gen_random_uuid()::text, $1, $2, NOW())`, *teamID, id); err != nil {
			return errx.Wrap(err, "open team membership", errx.TypeInternal)
		}
	}

	result, err := tx.ExecContext(ctx, `UPDATE enrollments SET team_id = $1 WHERE id = $2`, teamID, id)
	if err != nil {
		return errx.Wrap(err, "set enrollment team", errx.TypeInternal)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return enrollment.ErrNotFound()
	}

	if err := tx.Commit(); err != nil {
		return errx.Wrap(err, "commit set team", errx.TypeInternal)
	}
	return nil
}
