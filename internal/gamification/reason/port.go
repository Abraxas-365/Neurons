package reason

import (
	"context"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Repository defines the data access contract for Reason.
type Repository interface {
	Create(ctx context.Context, entity *Reason) error
	Update(ctx context.Context, entity *Reason) error
	GetByID(ctx context.Context, id kernel.ReasonID) (*Reason, error)
	// ListByClassroom returns the catalog; activeOnly powers the grant screen.
	ListByClassroom(ctx context.Context, classroomID kernel.ClassroomID, activeOnly bool) ([]Reason, error)
	Delete(ctx context.Context, id kernel.ReasonID) error
}
