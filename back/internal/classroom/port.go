package classroom

import (
	"context"
)

type DBRepository interface {
	CreateClassroom(ctx context.Context, classroom *Classroom) (*ClassroomWithData, error)
	GetClassroom(ctx context.Context, id int64) (*ClassroomWithData, error)
	UpdateClassroom(ctx context.Context, classroom *Classroom) (*ClassroomWithData, error)
	DeleteClassroom(ctx context.Context, id int64) error
	ListClassrooms(ctx context.Context, limit, offset int) ([]*ClassroomWithData, error)
	AddStudentToClassroom(ctx context.Context, classroomID, studentID int64) error
	RemoveStudentFromClassroom(ctx context.Context, classroomID, studentID int64) error
	UpdateAvailableNeurons(ctx context.Context, classroomID int64, neurons int) error
	GetClassroomStudents(ctx context.Context, classroomID int64) ([]*Student, error)
	IsStudentInClassroom(ctx context.Context, classroomID, studentID int64) (bool, error)
	TransferNeurons(ctx context.Context, classroomID, studentID int64, amount int) error
	RecordNeuronTransaction(ctx context.Context, transaction *NeuronTransaction) error
	GetUserNeurons(ctx context.Context, userID, classroomID int64) (int, error)
	ListUserClassrooms(ctx context.Context, userID int64, role string, limit, offset int) ([]*ClassroomWithData, error)
	TransferNeuronsToClassroom(ctx context.Context, classroomID, studentID int64, amount int) error
}
