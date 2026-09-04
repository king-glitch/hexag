module {{MODULE_PATH}}

go 1.26

require (
	github.com/gofiber/fiber/v3 v3.5.0
	github.com/joho/godotenv v1.5.1
	github.com/king-glitch/hexag v0.0.0
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.35.1
	go.mongodb.org/mongo-driver/v2 v2.8.0
)

replace github.com/king-glitch/hexag => {{HEXAG_PATH}}
