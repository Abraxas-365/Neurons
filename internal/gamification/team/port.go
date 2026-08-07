package team

import (
	"context"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Repository defines the data access contract for Team.
type Repository interface {
	Create(ctx context.Context, entity *Team) error
	Update(ctx context.Context, entity *Team) error
	GetByID(ctx context.Context, id kernel.TeamID) (*Team, error)
	ListByClassroom(ctx context.Context, classroomID kernel.ClassroomID) ([]Team, error)
	Delete(ctx context.Context, id kernel.TeamID) error

	// Members returns the team's active members with their balances.
	Members(ctx context.Context, id kernel.TeamID) ([]Member, error)
	// SetCoordinator marks one member as the team's coordinator.
	SetCoordinator(ctx context.Context, id kernel.TeamID, enrollmentID kernel.EnrollmentID) error
}
