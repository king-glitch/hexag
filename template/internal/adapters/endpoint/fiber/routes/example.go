package routes

import (
	"time"

	"{{MODULE_PATH}}/internal/ports"

	"github.com/gofiber/fiber/v3"
	hextransport "github.com/king-glitch/hexag/framework/httpx/transport"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ExampleHandler struct {
	service ports.ExampleService
}

func NewExampleHandler(service ports.ExampleService) ExampleHandler {
	return ExampleHandler{service: service}
}

func (h ExampleHandler) Register(router fiber.Router) {
	router.Get("/:id", h.get)
	router.Post("/", h.create)
}

func (h ExampleHandler) get(c fiber.Ctx) error {
	id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return errors.Wrap(err, "invalid example id")
	}

	example, serr := h.service.Get(c.RequestCtx(), id)
	if serr != nil {
		return serr
	}

	return hextransport.NewSuccessResponse(example).ToJSON(c)
}

type createExampleRequest struct {
	Name string `json:"name" validate:"required"`
}

func (h ExampleHandler) create(c fiber.Ctx) error {
	var req createExampleRequest
	if err := c.Bind().Body(&req); err != nil {
		return errors.Wrap(err, "failed to bind request")
	}

	example, serr := h.service.Create(c.RequestCtx(), req.Name, time.Now())
	if serr != nil {
		return serr
	}

	return hextransport.NewSuccessResponse(example).ToJSON(c)
}
