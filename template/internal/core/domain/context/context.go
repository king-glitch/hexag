package context

import (
	"github.com/rs/zerolog"

	"{{MODULE_PATH}}/internal/ports"

	hexports "github.com/king-glitch/hexag/framework/ports"
)

type Context struct {
	config            ports.Config
	logger            zerolog.Logger
	transactionRunner hexports.TransactionRunnerAdapter
}

func New(config ports.Config, logger zerolog.Logger, transactionRunner hexports.TransactionRunnerAdapter) ports.ServiceContext {
	return Context{
		config:            config,
		logger:            logger,
		transactionRunner: transactionRunner,
	}
}

func (c Context) Config() ports.Config {
	return c.config
}

func (c Context) GetConfig() ports.Config {
	return c.config
}

func (c Context) Logger() *zerolog.Logger {
	return &c.logger
}

func (c Context) GetLogger() *zerolog.Logger {
	return &c.logger
}

func (c Context) GetTransactionRunner() hexports.TransactionRunnerAdapter {
	return c.transactionRunner
}
