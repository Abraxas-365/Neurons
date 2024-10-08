package classroom

import (
	"strconv"

	"github.com/Abraxas-365/neurons/internal/user"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/Abraxas-365/toolkit/pkg/lucia"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service     Servicer
	userService user.Servicer
}

func NewHandler(service Servicer) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	classroomGroup := app.Group("/classrooms")

	// Routes that require authentication
	classroomGroup.Use(lucia.RequireAuth)
	classroomGroup.Post("/", h.CreateClassroom)
	classroomGroup.Get("/:id", h.GetClassroom)
	classroomGroup.Put("/:id", h.UpdateClassroom)
	classroomGroup.Delete("/:id", h.DeleteClassroom)
	classroomGroup.Get("/", h.ListClassrooms)
	classroomGroup.Post("/:id/students", h.AddStudentToClassroom)
	classroomGroup.Delete("/:id/students/:studentId", h.RemoveStudentFromClassroom)
	classroomGroup.Put("/:id/neurons", h.UpdateAvailableNeurons)
	classroomGroup.Get("/:id/students", h.GetClassroomStudents)
	classroomGroup.Post("/:id/send-neurons", h.SendNeurons)
	classroomGroup.Get("/:id/user-neurons/:userId", h.GetUserNeurons)
	classroomGroup.Get("/user", h.ListUserClassrooms)
	classroomGroup.Post("/:id/return-neurons", h.ReturnNeuronsToClassroom)
}

func (h *Handler) CreateClassroom(c *fiber.Ctx) error {
	session := lucia.GetSession(c)
	u, err := h.userService.GetUserByAuthUserID(c.Context(), session.UserID)
	if err != nil {
		return err
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return errors.ErrBadRequest("invalid input")
	}

	classroom, err := h.service.CreateClassRoom(c.Context(), u.ID, input.Name)
	if err != nil {
		return err
	}

	return c.JSON(classroom)
}

func (h *Handler) GetClassroom(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	classroom, err := h.service.GetClassroom(c.Context(), id)
	if err != nil {
		return err
	}

	return c.JSON(classroom)
}

func (h *Handler) UpdateClassroom(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	var input Classroom
	if err := c.BodyParser(&input); err != nil {
		return errors.ErrBadRequest("invalid input")
	}
	input.ID = id

	_, err = h.service.UpdateClassroom(c.Context(), &input)
	if err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) DeleteClassroom(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	err = h.service.DeleteClassroom(c.Context(), id)
	if err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) ListClassrooms(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	classrooms, err := h.service.ListClassrooms(c.Context(), limit, offset)
	if err != nil {
		return err
	}

	return c.JSON(classrooms)
}

func (h *Handler) AddStudentToClassroom(c *fiber.Ctx) error {
	classroomID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	var input struct {
		StudentID int64 `json:"student_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return errors.ErrBadRequest("invalid input")
	}

	err = h.service.AddStudentToClassroom(c.Context(), classroomID, input.StudentID)
	if err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) RemoveStudentFromClassroom(c *fiber.Ctx) error {
	classroomID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	studentID, err := strconv.ParseInt(c.Params("studentId"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid student id")
	}

	err = h.service.RemoveStudentFromClassroom(c.Context(), classroomID, studentID)
	if err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) UpdateAvailableNeurons(c *fiber.Ctx) error {
	classroomID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	var input struct {
		Neurons int `json:"neurons"`
	}
	if err := c.BodyParser(&input); err != nil {
		return errors.ErrBadRequest("invalid input")
	}

	err = h.service.UpdateAvailableNeurons(c.Context(), classroomID, input.Neurons)
	if err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) GetClassroomStudents(c *fiber.Ctx) error {
	classroomID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	students, err := h.service.GetClassroomStudents(c.Context(), classroomID)
	if err != nil {
		return err
	}

	return c.JSON(students)
}

func (h *Handler) SendNeurons(c *fiber.Ctx) error {
	session := lucia.GetSession(c)
	u, err := h.userService.GetUserByAuthUserID(c.Context(), session.UserID)
	if err != nil {
		return err
	}

	classroomID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	var input struct {
		StudentID int64 `json:"student_id"`
		Amount    int   `json:"amount"`
	}
	if err := c.BodyParser(&input); err != nil {
		return errors.ErrBadRequest("invalid input")
	}

	err = h.service.SendNeurons(c.Context(), u.ID, classroomID, input.StudentID, input.Amount)
	if err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *Handler) GetUserNeurons(c *fiber.Ctx) error {
	classroomID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid user id")
	}

	neurons, err := h.service.GetUserNeurons(c.Context(), userID, classroomID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"neurons": neurons})
}

func (h *Handler) ListUserClassrooms(c *fiber.Ctx) error {
	session := lucia.GetSession(c)
	u, err := h.userService.GetUserByAuthUserID(c.Context(), session.UserID)
	if err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	classrooms, err := h.service.ListUserClassrooms(c.Context(), u.ID, u.Role, limit, offset)
	if err != nil {
		return err
	}

	return c.JSON(classrooms)
}

func (h *Handler) ReturnNeuronsToClassroom(c *fiber.Ctx) error {
	session := lucia.GetSession(c)
	u, err := h.userService.GetUserByAuthUserID(c.Context(), session.UserID)
	if err != nil {
		return err
	}

	classroomID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return errors.ErrBadRequest("invalid classroom id")
	}

	var input struct {
		Amount int `json:"amount"`
	}
	if err := c.BodyParser(&input); err != nil {
		return errors.ErrBadRequest("invalid input")
	}

	err = h.service.ReturnNeuronsToClassroom(c.Context(), u.ID, classroomID, input.Amount)
	if err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}
