package context

import (
	"github.com/rs/zerolog"

	"{{MODULE_PATH}}/internal/ports"
)

type Context struct {
	config ports.Config
	logger zerolog.Logger
}

func New(config ports.Config, logger zerolog.Logger) ports.ServiceContext {
	return Context{
		config: config,
		logger: logger,
	}
}

func (c Context) Config() ports.Config {
	return c.config
}

func (c Context) Logger() *zerolog.Logger {
	return &c.logger
}
