package middleware

import (
	fiberzerolog "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

func NewLoggerMiddleware(logger *zerolog.Logger) fiber.Handler {
	return fiberzerolog.New(
		fiberzerolog.Config{
			Logger: logger,
		},
	)
}
