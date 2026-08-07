package enrollment

import (
	"context"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Repository defines the data access contract for Enrollment.
//
// Note: balance mutations are NOT part of this interface. Neuron movements go
// exclusively through the ledger module, which updates enrollments and the
// classroom vault inside one transaction (RN-09, RN-15).
type Repository interface {
	Create(ctx context.Context, entity *Enrollment) error
	Update(ctx context.Context, entity *Enrollment) error
	GetByID(ctx context.Context, id kernel.EnrollmentID) (*Enrollment, error)
	GetByUserAndClassroom(ctx context.Context, classroomID kernel.ClassroomID, userID kernel.UserID) (*Enrollment, error)

	// Roster returns the teacher-facing student list with filters (HU-023).
	Roster(ctx context.Context, classroomID kernel.ClassroomID, filter RosterFilter, opts kernel.PaginationOptions) (kernel.Paginated[Enrollment], error)

	// ListByUser returns every classroom a student participates in (HU-002).
	ListByUser(ctx context.Context, tenantID kernel.TenantID, userID kernel.UserID) ([]MyEnrollmentRow, error)

	// ListActiveIDsByTeam powers team grants (HU-033).
	ListActiveByTeam(ctx context.Context, teamID kernel.TeamID) ([]Enrollment, error)

	// ListActiveByIDs validates a multi-student selection (HU-052).
	ListActiveByIDs(ctx context.Context, classroomID kernel.ClassroomID, ids []kernel.EnrollmentID) ([]Enrollment, error)

	CountActive(ctx context.Context, classroomID kernel.ClassroomID) (int, error)

	// SetTeam assigns or clears a student's team and maintains the membership
	// history rows (decision 15.10).
	SetTeam(ctx context.Context, id kernel.EnrollmentID, teamID *kernel.TeamID) error
}
