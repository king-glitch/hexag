package routes

import (
	"{{MODULE_PATH}}/internal/ports"

	"github.com/gofiber/fiber/v3"
	hextransport "github.com/king-glitch/hexag/framework/api/http/transport"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ExampleHandler struct {
	es ports.ExampleService
}

func NewExampleHandler(es ports.ExampleService) ExampleHandler {
	return ExampleHandler{es: es}
}

func (h ExampleHandler) Register(router fiber.Router) {
	router.Get("/:id", h.Get)
	router.Post("/", h.Create)
}

type GetExampleResponse struct {
	ports.ExampleModel
}

func (h ExampleHandler) Get(c fiber.Ctx) error {
	id, err := bson.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return errors.Wrap(err, "invalid example id")
	}

	example, serr := h.es.Get(c.RequestCtx(), id)
	if serr != nil {
		return serr
	}

	return hextransport.NewSuccessResponse(GetExampleResponse{ExampleModel: example}).ToJSON(c)
}

type CreateExampleRequest struct {
	Name string `json:"name" validate:"required"`
}

type CreateExampleResponse struct {
	ports.ExampleModel
}

func (h ExampleHandler) Create(c fiber.Ctx) error {
	at := hextransport.RequestTime(c)

	var req CreateExampleRequest
	if err := hextransport.Bind(c, &req); err != nil {
		return errors.Wrap(err, "failed to bind request")
	}

	example, serr := h.es.Create(c.RequestCtx(), req.Name, at)
	if serr != nil {
		return serr
	}

	return hextransport.NewSuccessResponse(CreateExampleResponse{ExampleModel: example}).ToJSON(c)
}
