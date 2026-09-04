package fiber

import (
	"github.com/gofiber/fiber/v3"

	"{{MODULE_PATH}}/internal/adapters/endpoint/fiber/routes"
	"{{MODULE_PATH}}/internal/ports"

	hexhttpx "github.com/king-glitch/hexag/framework/httpx"
)

func New(
	serviceContext ports.ServiceContext,
	exampleHandler routes.ExampleHandler,
) *fiber.App {
	return hexhttpx.New(
		serviceContext.Logger(),
		func(api fiber.Router) {
			exampleHandler.Register(api.Group("/example"))
		},
	)
}
