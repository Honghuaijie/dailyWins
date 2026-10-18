package server

import (
	"dailyWins-service/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func NewGRPCServer(c *conf.Server, logger log.Logger) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.Middleware(
			requestLogMiddleware(logger),
			recovery.Recovery(),
		),
	}
	if c != nil && c.Grpc != nil {
		if c.Grpc.Network != "" {
			opts = append(opts, grpc.Network(c.Grpc.Network))
		}
		if c.Grpc.Addr != "" {
			opts = append(opts, grpc.Address(c.Grpc.Addr))
		}
		if c.Grpc.Timeout != nil {
			opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
		}
	}

	srv := grpc.NewServer(opts...)
	// V1 仅暴露认证后的 HTTP 契约；不注册模板用户 CRUD 或未鉴权 gRPC。
	return srv
}
