package user

import (
	"fmt"

	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/Abraxas-365/toolkit/pkg/lucia"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Servicer
}

func NewHandler(service Servicer) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	userGroup := app.Group("/users")

	// Routes that require authentication
	userGroup.Use(lucia.RequireAuth)
	userGroup.Get("/me", h.GetCurrentUser)
}

func (h *Handler) GetCurrentUser(c *fiber.Ctx) error {
	session := lucia.GetSession(c)
	if session == nil {
		return errors.ErrUnauthorized("No valid session found")
	}
	fmt.Println("session", session)

	user, err := h.service.GetUserByAuthUserID(c.Context(), session.UserID)
	if err != nil {
		return err
	}

	return c.JSON(user)
}
