package classroom

import (
	"context"
	"time"

	"github.com/Abraxas-365/neurons/internal/user"
	"github.com/Abraxas-365/toolkit/pkg/errors"
)

type Servicer interface {
	SendNeurons(ctx context.Context, teacherID, classroomID, studentID int64, amount int) error
	CreateClassRoom(ctx context.Context, teacherId int64, name string) (*ClassroomWithData, error)
	GetClassroom(ctx context.Context, id int64) (*ClassroomWithData, error)
	UpdateClassroom(ctx context.Context, classroom *Classroom) (*ClassroomWithData, error)
	DeleteClassroom(ctx context.Context, id int64) error
	ListClassrooms(ctx context.Context, limit, offset int) ([]*ClassroomWithData, error)
	AddStudentToClassroom(ctx context.Context, classroomID, studentID int64) error
	RemoveStudentFromClassroom(ctx context.Context, classroomID, studentID int64) error
	UpdateAvailableNeurons(ctx context.Context, classroomID int64, neurons int) error
	GetClassroomStudents(ctx context.Context, classroomID int64) ([]*Student, error)
	GetUserNeurons(ctx context.Context, userID, classroomID int64) (int, error)
	ListUserClassrooms(ctx context.Context, userID int64, role string, limit, offset int) ([]*ClassroomWithData, error)
	ReturnNeuronsToClassroom(ctx context.Context, studentID, classroomID int64, amount int) error
}

var _ Servicer = (*Service)(nil)

type Service struct {
	userService user.Servicer
	repo        DBRepository
}

// NewService creates a new classroom service
func NewService(userService user.Servicer, repo DBRepository) *Service {
	return &Service{
		userService: userService,
		repo:        repo,
	}
}

// CreateClassRoom creates a new classroom
func (s *Service) CreateClassRoom(ctx context.Context, teacherId int64, name string) (*ClassroomWithData, error) {
	// Verify that the teacher exists
	teacher, err := s.userService.GetUser(ctx, teacherId)
	if err != nil {
		return nil, err
	}

	if teacher.Role != "teacher" {
		return nil, errors.ErrBadRequest("user is not a teacher")
	}

	classroom := &Classroom{
		Name:             name,
		TeacherID:        teacherId,
		AvailableNeurons: 0, // Initial value, can be changed if needed
		CreatedAt:        time.Now(),
	}

	classroomWithData, err := s.repo.CreateClassroom(ctx, classroom)
	if err != nil {
		return nil, err
	}

	return classroomWithData, nil
}

// GetClassroom retrieves a classroom by ID
func (s *Service) GetClassroom(ctx context.Context, id int64) (*ClassroomWithData, error) {
	classroomWithData, err := s.repo.GetClassroom(ctx, id)
	if err != nil {
		return nil, err
	}
	return classroomWithData, nil
}

// UpdateClassroom updates an existing classroom
func (s *Service) UpdateClassroom(ctx context.Context, classroom *Classroom) (*ClassroomWithData, error) {
	classroomWithData, err := s.repo.UpdateClassroom(ctx, classroom)
	if err != nil {
		return nil, err
	}
	return classroomWithData, nil
}

// DeleteClassroom deletes a classroom
func (s *Service) DeleteClassroom(ctx context.Context, id int64) error {
	err := s.repo.DeleteClassroom(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

// ListClassrooms retrieves a list of classrooms
func (s *Service) ListClassrooms(ctx context.Context, limit, offset int) ([]*ClassroomWithData, error) {
	classroomsWithData, err := s.repo.ListClassrooms(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return classroomsWithData, nil
}

// AddStudentToClassroom adds a student to a classroom
func (s *Service) AddStudentToClassroom(ctx context.Context, classroomID, studentID int64) error {
	// Verify that the student exists
	student, err := s.userService.GetUser(ctx, studentID)
	if err != nil {
		return err
	}

	if student.Role != "student" {
		return errors.ErrBadRequest("user is not a student")
	}

	err = s.repo.AddStudentToClassroom(ctx, classroomID, studentID)
	if err != nil {
		return err
	}
	return nil
}

// RemoveStudentFromClassroom removes a student from a classroom
func (s *Service) RemoveStudentFromClassroom(ctx context.Context, classroomID, studentID int64) error {
	err := s.repo.RemoveStudentFromClassroom(ctx, classroomID, studentID)
	if err != nil {
		return err
	}
	return nil
}

// UpdateAvailableNeurons updates the available neurons for a classroom
func (s *Service) UpdateAvailableNeurons(ctx context.Context, classroomID int64, neurons int) error {
	err := s.repo.UpdateAvailableNeurons(ctx, classroomID, neurons)
	if err != nil {
		return err
	}
	return nil
}

// GetClassroomStudents retrieves all students in a classroom
func (s *Service) GetClassroomStudents(ctx context.Context, classroomID int64) ([]*Student, error) {
	students, err := s.repo.GetClassroomStudents(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	return students, nil
}

func (s *Service) SendNeurons(ctx context.Context, teacherID, classroomID, studentID int64, amount int) error {
	// Verify that the sender is a teacher
	teacher, err := s.userService.GetUser(ctx, teacherID)
	if err != nil {
		return err
	}
	if teacher.Role != "teacher" {
		return errors.ErrForbidden("only teachers can send neurons")
	}

	// Verify that the classroom belongs to the teacher
	classroom, err := s.repo.GetClassroom(ctx, classroomID)
	if err != nil {
		return err
	}
	if classroom.TeacherID != teacherID {
		return errors.ErrForbidden("teacher does not own this classroom")
	}

	// Verify that the student is in the classroom
	isStudentInClassroom, err := s.repo.IsStudentInClassroom(ctx, classroomID, studentID)
	if err != nil {
		return err
	}
	if !isStudentInClassroom {
		return errors.ErrBadRequest("student is not in this classroom")
	}

	// Verify that the classroom has enough neurons
	if classroom.AvailableNeurons < amount {
		return errors.ErrBadRequest("not enough neurons in the classroom")
	}

	// Perform the neuron transfer
	err = s.repo.TransferNeurons(ctx, classroomID, studentID, amount)
	if err != nil {
		return err
	}

	// Record the transaction
	transaction := &NeuronTransaction{
		ClassroomID:     classroomID,
		UserID:          studentID,
		Amount:          amount,
		TransactionType: "assignment",
		CreatedAt:       time.Now(),
	}
	err = s.repo.RecordNeuronTransaction(ctx, transaction)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) GetUserNeurons(ctx context.Context, userID, classroomID int64) (int, error) {
	// Verify that the user exists
	_, err := s.userService.GetUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Verify that the classroom exists
	_, err = s.repo.GetClassroom(ctx, classroomID)
	if err != nil {
		return 0, err
	}

	// Get the user's neurons for this classroom
	neurons, err := s.repo.GetUserNeurons(ctx, userID, classroomID)
	if err != nil {
		return 0, err
	}

	return neurons, nil
}

func (s *Service) ListUserClassrooms(ctx context.Context, userID int64, role string, limit, offset int) ([]*ClassroomWithData, error) {
	// Verify that the user exists and has the correct role
	user, err := s.userService.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Role != role {
		return nil, errors.ErrBadRequest("user role does not match the requested role")
	}

	return s.repo.ListUserClassrooms(ctx, userID, role, limit, offset)
}

func (s *Service) ReturnNeuronsToClassroom(ctx context.Context, studentID, classroomID int64, amount int) error {
	// Verify that the user exists and is a student
	student, err := s.userService.GetUser(ctx, studentID)
	if err != nil {
		return err
	}
	if student.Role != "student" {
		return errors.ErrForbidden("only students can return neurons")
	}

	// Verify that the classroom exists
	_, err = s.repo.GetClassroom(ctx, classroomID)
	if err != nil {
		return err
	}

	// Verify that the student is in the classroom
	isStudentInClassroom, err := s.repo.IsStudentInClassroom(ctx, classroomID, studentID)
	if err != nil {
		return err
	}
	if !isStudentInClassroom {
		return errors.ErrBadRequest("student is not in this classroom")
	}

	// Verify that the student has enough neurons to return
	studentNeurons, err := s.repo.GetUserNeurons(ctx, studentID, classroomID)
	if err != nil {
		return err
	}
	if studentNeurons < amount {
		return errors.ErrBadRequest("student does not have enough neurons to return")
	}

	// Perform the neuron transfer (from student back to classroom)
	err = s.repo.TransferNeuronsToClassroom(ctx, classroomID, studentID, amount)
	if err != nil {
		return err
	}

	// Record the transaction
	transaction := &NeuronTransaction{
		ClassroomID:     classroomID,
		UserID:          studentID,
		Amount:          amount,
		TransactionType: "return",
		CreatedAt:       time.Now(),
	}
	err = s.repo.RecordNeuronTransaction(ctx, transaction)
	if err != nil {
		return err
	}

	return nil
}
