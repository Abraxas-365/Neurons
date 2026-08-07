package benefit

import (
	"context"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Repository defines the data access contract for Benefit.
type Repository interface {
	Create(ctx context.Context, entity *Benefit) error
	Update(ctx context.Context, entity *Benefit) error
	GetByID(ctx context.Context, id kernel.BenefitID) (*Benefit, error)
	ListByClassroom(ctx context.Context, classroomID kernel.ClassroomID, activeOnly bool) ([]Benefit, error)
	Delete(ctx context.Context, id kernel.BenefitID) error

	// CountUsesByStudent supports the per-student quota check.
	CountUsesByStudent(ctx context.Context, id kernel.BenefitID, enrollmentID kernel.EnrollmentID) (int, error)
}
