// Package httpx holds small helpers shared by the gamification HTTP handlers.
package httpx

import (
	"net/http"

	"github.com/Abraxas-356/neurons/internal/errx"
	"github.com/Abraxas-356/neurons/internal/iam/auth"
	"github.com/Abraxas-356/neurons/internal/kernel"
	"github.com/gofiber/fiber/v2"
)

var ErrRegistry = errx.NewRegistry("HTTP")

var (
	CodeUnauthorized = ErrRegistry.Register(
		"UNAUTHORIZED",
		errx.TypeAuthorization,
		http.StatusUnauthorized,
		"Authentication required",
	)

	CodeBadRequest = ErrRegistry.Register(
		"BAD_REQUEST",
		errx.TypeValidation,
		http.StatusBadRequest,
		"Invalid request",
	)
)

func ErrUnauthorized() error { return ErrRegistry.New(CodeUnauthorized) }

func ErrBadRequest(msg string) error {
	return ErrRegistry.NewWithMessage(CodeBadRequest, msg)
}

// Actor is the authenticated caller, reduced to what the domain services need.
type Actor struct {
	UserID   kernel.UserID
	TenantID kernel.TenantID
	Email    string
	Name     string
}

// CurrentActor extracts the authenticated user from the request context.
func CurrentActor(c *fiber.Ctx) (Actor, error) {
	authCtx, ok := auth.GetAuthContext(c)
	if !ok || authCtx.UserID == nil {
		return Actor{}, ErrUnauthorized()
	}
	return Actor{
		UserID:   *authCtx.UserID,
		TenantID: authCtx.TenantID,
		Email:    authCtx.Email,
		Name:     authCtx.Name,
	}, nil
}

// BodyParser decodes the JSON body into dst, returning a domain error on failure.
func BodyParser[T any](c *fiber.Ctx, dst *T) error {
	if err := c.BodyParser(dst); err != nil {
		return ErrBadRequest("invalid request body")
	}
	return nil
}

// Pagination reads page/page_size query params with sane bounds.
func Pagination(c *fiber.Ctx) kernel.PaginationOptions {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	size := c.QueryInt("page_size", 20)
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return kernel.PaginationOptions{Page: page, PageSize: size}
}

// ClassroomID reads a classroom id from the given route param.
func ClassroomID(c *fiber.Ctx, param string) (kernel.ClassroomID, error) {
	raw := c.Params(param)
	if raw == "" {
		return "", ErrBadRequest(param + " is required")
	}
	return kernel.NewClassroomID(raw), nil
}
