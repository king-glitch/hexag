package httpx

import (
	"encoding/json/v2"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/king-glitch/hexag/framework/httpx/middleware"
	"github.com/king-glitch/hexag/framework/httpx/transport"
	"github.com/king-glitch/hexag/framework/httpx/validator"
	"github.com/king-glitch/hexag/framework/ports"
)

// New builds a *fiber.App wired with CORS, request logging, the shared
// struct validator and error handler. mount is called with the "/api/v1"
// group so the project registers its own route handlers — route
// registration is business surface, not infrastructure, so it never moves
// into the framework.
func New(logger *zerolog.Logger, mount func(api fiber.Router)) *fiber.App {
	app := fiber.New(
		fiber.Config{
			JSONEncoder: func(v any) ([]byte, error) {
				return json.Marshal(v)
			},
			JSONDecoder: func(data []byte, v any) error {
				return json.Unmarshal(data, v)
			},
			Immutable:       true,
			StructValidator: validator.NewStructValidator(),
			ErrorHandler:    errorHandler,
		},
	)

	app.Use(func(c fiber.Ctx) error {
		c.Locals("server_time", time.Now().UTC())
		return c.Next()
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}))
	app.Use(middleware.NewLoggerMiddleware(logger))

	mount(app.Group("/api/v1"))

	return app
}

func errorHandler(c fiber.Ctx, err error) error {
	statusCode := fiber.StatusInternalServerError

	var fiberErr *fiber.Error
	var serviceErr *ports.ServiceError

	switch {
	case errors.As(err, &serviceErr):
		statusCode = serviceErr.Code.DefaultStatusCode()
		if serviceErr.StatusCode != 0 {
			statusCode = serviceErr.StatusCode
		}
	case errors.As(err, &fiberErr):
		statusCode = fiberErr.Code
		serviceErr = ports.NewServiceError(ports.ServiceErrorCodeInternal, errors.New(fiberErr.Message))
	default:
		serviceErr = ports.NewServiceError(ports.ServiceErrorCodeInternal, err)
	}

	if err := c.Status(statusCode).JSON(
		transport.Response{
			Data:   fiber.Map{},
			Errors: serviceErr,
		},
	); err != nil {
		return errors.Wrap(err, "failed to send error response")
	}

	return nil
}

// ParsePaginationParamsContext parses standard pagination parameters from fiber.Ctx.
func ParsePaginationParamsContext(ctx fiber.Ctx, defaultAmount ...int) ports.PaginationParams {
	return transport.ParsePaginationParamsContext(ctx, defaultAmount...)
}

// ParsePaginationParams is an alias for ParsePaginationParamsContext.
func ParsePaginationParams(ctx fiber.Ctx, defaultAmount ...int) ports.PaginationParams {
	return transport.ParsePaginationParamsContext(ctx, defaultAmount...)
}

// Bind binds query parameters on GET/DELETE/HEAD requests, and JSON body + query parameters on other methods.
func Bind(ctx fiber.Ctx, out any) error {
	return transport.Bind(ctx, out)
}

// BindQuery binds URL query parameters into the struct pointed to by out.
func BindQuery(ctx fiber.Ctx, out any) error {
	return transport.BindQuery(ctx, out)
}

// RequestTime returns the request's server time (in UTC).
func RequestTime(ctx fiber.Ctx) time.Time {
	return transport.RequestTime(ctx)
}
