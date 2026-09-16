package base

import (
	"github.com/king-glitch/hexag/framework/ports"
)

type ServiceBase struct {
	context ports.ServiceContext
}

func NewBaseService(ctx ports.ServiceContext) ServiceBase {
	return ServiceBase{
		context: ctx,
	}
}

func (s ServiceBase) GetContext() ports.ServiceContext {
	return s.context
}

func (s ServiceBase) GetTransactionRunner() ports.TransactionRunnerAdapter {
	return s.context.GetTransactionRunner()
}
