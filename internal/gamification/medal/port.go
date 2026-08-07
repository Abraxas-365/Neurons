package medal

import (
	"context"

	"github.com/Abraxas-356/neurons/internal/kernel"
)

// Repository defines the data access contract for Medal and its awards.
type Repository interface {
	Create(ctx context.Context, entity *Medal) error
	Update(ctx context.Context, entity *Medal) error
	GetByID(ctx context.Context, id kernel.MedalID) (*Medal, error)
	ListByClassroom(ctx context.Context, classroomID kernel.ClassroomID, activeOnly bool) ([]Medal, error)
	Delete(ctx context.Context, id kernel.MedalID) error

	// AwardMany inserts all awards of one operation atomically (HU-072).
	AwardMany(ctx context.Context, awards []Award) error
	GetAward(ctx context.Context, awardID string) (*Award, error)
	// RevokeAward soft-deletes an award (RN-15: history is never erased).
	RevokeAward(ctx context.Context, awardID string) error

	CountAwards(ctx context.Context, id kernel.MedalID) (int, error)
	HasAward(ctx context.Context, id kernel.MedalID, enrollmentID *kernel.EnrollmentID, teamID *kernel.TeamID) (bool, error)

	ListAwardsByClassroom(ctx context.Context, classroomID kernel.ClassroomID) ([]Award, error)
	// ListAwardsForStudent returns the student's own medals plus, when the medal
	// is configured to show on member profiles, their team medals (RN-14).
	ListAwardsForStudent(ctx context.Context, enrollmentID kernel.EnrollmentID) ([]Award, error)
	ListAwardsByTeam(ctx context.Context, teamID kernel.TeamID) ([]Award, error)
	CountByEnrollment(ctx context.Context, enrollmentID kernel.EnrollmentID) (int, error)
}
