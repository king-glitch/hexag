package transport

import (
	"github.com/gofiber/fiber/v3"
	"github.com/pkg/errors"

	"github.com/king-glitch/hexag/framework/ports"
)

type Response struct {
	Data   any                 `json:"data"`
	Errors *ports.ServiceError `json:"errors,omitempty"`
}

func (r Response) ToJSON(c fiber.Ctx) error {
	if err := c.JSON(r); err != nil {
		return errors.Wrap(err, "failed to serialize json response")
	}

	return nil
}

func NewSuccessResponse(data ...any) Response {
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	return Response{Data: d}
}
