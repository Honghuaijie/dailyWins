package biz

import (
	"context"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewUserUsecase, NewTaskUsecase, NewAccountUsecase)

type Transaction interface {
	ExecTx(context.Context, func(ctx context.Context) error) error
}
