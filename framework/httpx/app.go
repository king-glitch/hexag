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

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	AllowOrigins     []string
	AllowHeaders     []string
	AllowMethods     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// Config holds configuration for the httpx HTTP application.
type Config struct {
	CORS CORSConfig
}

// Option configures an httpx application.
type Option func(*Config)

// WithConfig sets the entire httpx Config.
func WithConfig(cfg Config) Option {
	return func(c *Config) {
		*c = cfg
	}
}

// WithCORS sets the CORS configuration.
func WithCORS(cors CORSConfig) Option {
	return func(c *Config) {
		c.CORS = cors
	}
}

// WithAllowOrigins sets allowed CORS origins.
func WithAllowOrigins(origins ...string) Option {
	return func(c *Config) {
		c.CORS.AllowOrigins = origins
	}
}

// WithAllowHeaders sets allowed CORS request headers.
func WithAllowHeaders(headers ...string) Option {
	return func(c *Config) {
		c.CORS.AllowHeaders = headers
	}
}

// DefaultCORSConfig returns standard default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}
}

// DefaultConfig returns the default httpx configuration.
func DefaultConfig() Config {
	return Config{
		CORS: DefaultCORSConfig(),
	}
}

// New builds a *fiber.App wired with CORS, request logging, the shared
// struct validator and error handler. mount is called with the "/api/v1"
// group so the project registers its own route handlers — route
// registration is business surface, not infrastructure, so it never moves
// into the framework.
func New(logger *zerolog.Logger, mount func(api fiber.Router), opts ...Option) *fiber.App {
	cfg := DefaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	applyConfigDefaults(&cfg)

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
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowHeaders:     cfg.CORS.AllowHeaders,
		AllowMethods:     cfg.CORS.AllowMethods,
		ExposeHeaders:    cfg.CORS.ExposeHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))
	app.Use(middleware.NewLoggerMiddleware(logger))

	mount(app.Group("/api/v1"))

	return app
}

func applyConfigDefaults(cfg *Config) {
	if len(cfg.CORS.AllowOrigins) == 0 {
		cfg.CORS.AllowOrigins = []string{"*"}
	}
	if len(cfg.CORS.AllowHeaders) == 0 {
		cfg.CORS.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	}
	if len(cfg.CORS.AllowMethods) == 0 {
		cfg.CORS.AllowMethods = []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH", "OPTIONS"}
	}
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
