package classroom

import (
	"context"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Repository defines the data access contract for Classroom.
type Repository interface {
	Create(ctx context.Context, entity *Classroom) error
	Update(ctx context.Context, entity *Classroom) error
	GetByID(ctx context.Context, id kernel.ClassroomID) (*Classroom, error)
	GetByInviteCode(ctx context.Context, code string) (*Classroom, error)
	List(ctx context.Context, tenantID kernel.TenantID, opts kernel.PaginationOptions) (kernel.Paginated[Classroom], error)
	// ListForTeacher returns classrooms where the user teaches (owner or assistant).
	ListForTeacher(ctx context.Context, tenantID kernel.TenantID, userID kernel.UserID, opts kernel.PaginationOptions) (kernel.Paginated[Classroom], error)
	Delete(ctx context.Context, id kernel.ClassroomID) error

	// --- Teachers ---
	AddTeacher(ctx context.Context, t *ClassroomTeacher) error
	RemoveTeacher(ctx context.Context, id kernel.ClassroomID, userID kernel.UserID) error
	GetTeacher(ctx context.Context, id kernel.ClassroomID, userID kernel.UserID) (*ClassroomTeacher, error)
	ListTeachers(ctx context.Context, id kernel.ClassroomID) ([]ClassroomTeacher, error)
}
