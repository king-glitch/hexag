package ports

import "github.com/rs/zerolog"

// ServiceContext provides the baseline infrastructure dependencies required by
// services: a logger and a transaction runner. A project's own ServiceContext
// embeds this and adds its own Config(), since config shape is project-specific
// (env vars differ per project) and the framework has no business knowing it.
type ServiceContext interface {
	Logger() *zerolog.Logger
	GetLogger() *zerolog.Logger
	GetTransactionRunner() TransactionRunnerAdapter
}

type ServiceBase interface {
	GetContext() ServiceContext
}
