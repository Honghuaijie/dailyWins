package server

import (
	"context"
	"time"

	httpErrors "dailyWins-service/internal/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

func requestLogMiddleware(logger log.Logger) middleware.Middleware {
	helper := log.NewHelper(logger)
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			start := time.Now()
			reply, err = next(ctx, req)
			latency := time.Since(start)

			transportKind := ""
			operation := ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				transportKind = tr.Kind().String()
				operation = tr.Operation()
			}

			fields := []interface{}{
				"component", "server_middleware",
				"transport", transportKind,
				"operation", operation,
				"latency_ms", latency.Milliseconds(),
			}

			if err != nil {
				se := httpErrors.FromError(err)
				fields = append(fields,
					"code", se.Code,
					"http_code", se.HttpCode,
					"err", err.Error(),
				)
				helper.Errorw(fields...)
				return reply, err
			}

			fields = append(fields, "code", 0)
			helper.Infow(fields...)
			return reply, nil
		}
	}
}
