package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	exampleadapter "{{MODULE_PATH}}/internal/adapters/database/mongo/repository/example"
	fiberadapter "{{MODULE_PATH}}/internal/adapters/endpoint/fiber"
	"{{MODULE_PATH}}/internal/adapters/endpoint/fiber/routes"
	domaincontext "{{MODULE_PATH}}/internal/core/domain/context"
	exampleservice "{{MODULE_PATH}}/internal/services/example"
)

func main() {
	logger := zerolog.New(
		&zerolog.ConsoleWriter{
			Out: os.Stderr,
		},
	).With().Timestamp().Logger()

	if err := godotenv.Load(); err != nil {
		logger.Info().Msg("no .env file found, reading config from process environment")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := domaincontext.LoadConfigFromEnv()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	serviceContext := domaincontext.New(config, logger)

	client, err := mongo.Connect(options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to mongo")
	}

	db := client.Database(config.MongoDatabase)

	exampleRepository := exampleadapter.NewRepository(db)
	if err := exampleadapter.EnsureIndexes(ctx, db); err != nil {
		logger.Fatal().Err(err).Msg("failed to ensure example indexes")
	}

	es := exampleservice.NewService(serviceContext, exampleRepository)

	app := fiberadapter.New(serviceContext, routes.NewExampleHandler(es))

	serverErrors := make(chan error, 1)

	go func() {
		if err := app.Listen(":" + config.Port); err != nil {
			serverErrors <- err
		}
		close(serverErrors)
	}()

	logger.Info().Str("port", config.Port).Msg("server started")

	select {
	case err, ok := <-serverErrors:
		if ok && err != nil {
			logger.Error().Err(err).Msg("server stopped unexpectedly")
		}
	case <-ctx.Done():
		logger.Info().Msg("shutdown signal received")
	}

	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("failed to gracefully shut down http server")
	} else {
		logger.Info().Msg("http server shut down")
	}

	if err := client.Disconnect(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("failed to disconnect mongo client")
	} else {
		logger.Info().Msg("mongo client disconnected")
	}

	logger.Info().Msg("shutdown complete")
}
