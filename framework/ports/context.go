package ports

import "github.com/rs/zerolog"

// ServiceContext is the minimum every framework service needs: a logger. A
// project's own ServiceContext embeds this and adds its own Config(), since
// config shape is project-specific (env vars differ per project) and the
// framework has no business knowing it.
type ServiceContext interface {
	Logger() *zerolog.Logger
}

type ServiceBase struct {
	Context ServiceContext
}

func NewBaseService(ctx ServiceContext) ServiceBase {
	return ServiceBase{
		Context: ctx,
	}
}

func (s ServiceBase) GetContext() ServiceContext {
	return s.Context
}
