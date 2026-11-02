package httpx_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	hexhttpx "github.com/king-glitch/hexag/framework/api/http"
)

func TestNew_DefaultConfig(t *testing.T) {
	logger := zerolog.Nop()
	app := hexhttpx.New(&logger, func(api fiber.Router) {
		api.Get("/ping", func(c fiber.Ctx) error {
			return c.SendString("pong")
		})
	})

	// Preflight OPTIONS check for default allowed headers
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/ping", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	require.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Content-Type")
	require.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestNew_WithCORS(t *testing.T) {
	logger := zerolog.Nop()
	app := hexhttpx.New(
		&logger,
		func(api fiber.Router) {
			api.Post("/booking/requests", func(c fiber.Ctx) error {
				return c.SendString("ok")
			})
		},
		hexhttpx.WithCORS(hexhttpx.CORSConfig{
			AllowOrigins: []string{"http://localhost:5173"},
			AllowHeaders: []string{
				"Origin",
				"Content-Type",
				"Accept",
				"Authorization",
				"Idempotency-Key",
				"X-Requested-With",
			},
		}),
	)

	// Preflight OPTIONS check for custom headers including Idempotency-Key
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/booking/requests", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Idempotency-Key")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Equal(t, "http://localhost:5173", resp.Header.Get("Access-Control-Allow-Origin"))
	require.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Idempotency-Key")
	require.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "X-Requested-With")
}

func TestNew_WithAllowHeaders(t *testing.T) {
	logger := zerolog.Nop()
	app := hexhttpx.New(
		&logger,
		func(api fiber.Router) {
			api.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("ok")
			})
		},
		hexhttpx.WithAllowHeaders("Content-Type", "Idempotency-Key"),
	)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "Idempotency-Key")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Idempotency-Key")
}

func TestNew_WithConfig(t *testing.T) {
	logger := zerolog.Nop()
	app := hexhttpx.New(
		&logger,
		func(api fiber.Router) {
			api.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("ok")
			})
		},
		hexhttpx.WithConfig(hexhttpx.Config{
			CORS: hexhttpx.CORSConfig{
				AllowOrigins: []string{"*"},
				AllowHeaders: []string{"Authorization", "X-Custom-Header"},
			},
		}),
	)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "X-Custom-Header")
}

func TestNew_WithBodyLimit(t *testing.T) {
	logger := zerolog.Nop()
	app := hexhttpx.New(
		&logger,
		func(api fiber.Router) {
			api.Post("/upload", func(c fiber.Ctx) error {
				return c.SendString(string(c.Body()))
			})
		},
		hexhttpx.WithBodyLimit(10),
	)

	// Within limit: 5 bytes
	reqSmall := httptest.NewRequest(http.MethodPost, "/api/v1/upload", bytes.NewReader([]byte("hello")))
	respSmall, err := app.Test(reqSmall)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, respSmall.StatusCode)

	// Over limit: 20 bytes
	reqLarge := httptest.NewRequest(http.MethodPost, "/api/v1/upload", bytes.NewReader([]byte("12345678901234567890")))
	respLarge, err := app.Test(reqLarge)
	if err != nil {
		require.Contains(t, err.Error(), "body size exceeds")
	} else {
		require.Equal(t, http.StatusRequestEntityTooLarge, respLarge.StatusCode)
	}
}

func TestNew_WithStreamRequestBody(t *testing.T) {
	logger := zerolog.Nop()
	app := hexhttpx.New(
		&logger,
		func(api fiber.Router) {
			api.Post("/stream", func(c fiber.Ctx) error {
				return c.SendString("streamed")
			})
		},
		hexhttpx.WithStreamRequestBody(true),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stream", bytes.NewReader([]byte("test data")))
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
