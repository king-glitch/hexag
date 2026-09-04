package transport_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"

	"github.com/king-glitch/hexag/framework/httpx/transport"
)

func TestParsePaginationParamsContext(t *testing.T) {
	app := fiber.New()

	app.Get("/test-default", func(c fiber.Ctx) error {
		params := transport.ParsePaginationParamsContext(c)
		assert.Equal(t, 1, params.Page)
		assert.Equal(t, 10, params.Amount)
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/test-custom-default", func(c fiber.Ctx) error {
		params := transport.ParsePaginationParamsContext(c, 20)
		assert.Equal(t, 1, params.Page)
		assert.Equal(t, 20, params.Amount)
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/test-query", func(c fiber.Ctx) error {
		params := transport.ParsePaginationParamsContext(c, 20)
		assert.Equal(t, 3, params.Page)
		assert.Equal(t, 50, params.Amount)
		return c.SendStatus(fiber.StatusOK)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/test-default", nil)
	_, err := app.Test(req1)
	assert.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodGet, "/test-custom-default", nil)
	_, err = app.Test(req2)
	assert.NoError(t, err)

	req3 := httptest.NewRequest(http.MethodGet, "/test-query?page=3&amount=50", nil)
	_, err = app.Test(req3)
	assert.NoError(t, err)
}
