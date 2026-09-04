package transport_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/king-glitch/hexag/framework/httpx/transport"
	"github.com/king-glitch/hexag/framework/ports"
)

type CustomAction string

type TestFilter struct {
	Actions     []CustomAction  `query:"action"`
	BookingID   bson.ObjectID   `query:"booking_id"`
	PropertyID  *bson.ObjectID  `query:"property_id"`
	RoomIDs     []bson.ObjectID `query:"room_ids"`
	From        *time.Time      `query:"from"`
	To          time.Time       `query:"to"`
	Status      string          `query:"status"`
	MaxCount    int             `query:"max_count"`
	IsActive    bool            `query:"is_active"`
}

func TestBindQuery_Success(t *testing.T) {
	app := fiber.New()

	bookingID := bson.NewObjectID()
	propertyID := bson.NewObjectID()
	roomID1 := bson.NewObjectID()
	roomID2 := bson.NewObjectID()

	app.Get("/test-filter", func(c fiber.Ctx) error {
		var filter TestFilter
		err := transport.BindQuery(c, &filter)
		assert.NoError(t, err)

		assert.Equal(t, 2, len(filter.Actions))
		assert.Equal(t, CustomAction("booking.confirmed"), filter.Actions[0])
		assert.Equal(t, CustomAction("booking.created"), filter.Actions[1])

		assert.Equal(t, bookingID, filter.BookingID)
		assert.NotNil(t, filter.PropertyID)
		assert.Equal(t, propertyID, *filter.PropertyID)

		assert.Equal(t, 2, len(filter.RoomIDs))
		assert.Equal(t, roomID1, filter.RoomIDs[0])
		assert.Equal(t, roomID2, filter.RoomIDs[1])

		assert.NotNil(t, filter.From)
		assert.Equal(t, 2026, filter.From.Year())
		assert.Equal(t, time.Month(8), filter.From.Month())
		assert.Equal(t, 1, filter.From.Day())

		assert.Equal(t, 2026, filter.To.Year())
		assert.Equal(t, time.Month(8), filter.To.Month())
		assert.Equal(t, 15, filter.To.Day())

		assert.Equal(t, "confirmed", filter.Status)
		assert.Equal(t, 42, filter.MaxCount)
		assert.True(t, filter.IsActive)

		return c.SendStatus(fiber.StatusOK)
	})

	url := "/test-filter?action=booking.confirmed,booking.created" +
		"&booking_id=" + bookingID.Hex() +
		"&property_id=" + propertyID.Hex() +
		"&room_ids=" + roomID1.Hex() + "," + roomID2.Hex() +
		"&from=2026-08-01T00:00:00Z" +
		"&to=2026-08-15" +
		"&status=confirmed" +
		"&max_count=42" +
		"&is_active=true"

	req := httptest.NewRequest(http.MethodGet, url, nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBindQuery_InvalidObjectID(t *testing.T) {
	app := fiber.New()

	var bindErr error
	app.Get("/test-invalid-id", func(c fiber.Ctx) error {
		var filter TestFilter
		bindErr = transport.BindQuery(c, &filter)
		return bindErr
	})

	req := httptest.NewRequest(http.MethodGet, "/test-invalid-id?booking_id=invalid-hex", nil)
	_, _ = app.Test(req)
	assert.Error(t, bindErr)

	var serr *ports.ServiceError
	assert.True(t, errors.As(bindErr, &serr))
	assert.Equal(t, ports.ServiceErrorCodeValidation, serr.Code)
}

type DefaultsFilter struct {
	Adults int    `query:"adults" default:"2"`
	Rooms  int    `query:"rooms" default:"1"`
	Status string `query:"status" default:"active"`
}

func TestBindQuery_Defaults(t *testing.T) {
	app := fiber.New()

	app.Get("/test-defaults", func(c fiber.Ctx) error {
		var filter DefaultsFilter
		err := transport.BindQuery(c, &filter)
		assert.NoError(t, err)

		assert.Equal(t, 2, filter.Adults)
		assert.Equal(t, 1, filter.Rooms)
		assert.Equal(t, "active", filter.Status)

		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test-defaults", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

