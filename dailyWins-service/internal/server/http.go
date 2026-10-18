package server

import (
	v1 "dailyWins-service/api/all-pb-go/v1"
	"dailyWins-service/internal/auth"
	"dailyWins-service/internal/conf"
	"dailyWins-service/internal/handler"
	"dailyWins-service/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(c *conf.Server, dailySvc *service.DailyWinsService, authOptions *auth.Options, logger log.Logger) *http.Server {
	opts := []http.ServerOption{
		// 给kratos的http服务注册一个“统一错误处理函数”--只要业务层返回err，都会调用这个方法
		http.ErrorEncoder(errorEncoder),
		http.RequestDecoder(decodeJSON),
		http.Middleware(
			requestLogMiddleware(logger),
			recovery.Recovery(),
			validate.Validator(),
		),
		http.Filter(securityFilter(authOptions)),
	}
	if c != nil && c.Http != nil {
		if c.Http.Network != "" {
			opts = append(opts, http.Network(c.Http.Network))
		}
		if c.Http.Addr != "" {
			opts = append(opts, http.Address(c.Http.Addr))
		}
		if c.Http.Timeout != nil {
			opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
		}
	}

	srv := http.NewServer(opts...)
	v1.RegisterDailyWinsHTTPServer(srv, dailySvc)
	srv.Route("").GET("/health", handler.HealthCheckHandler)
	log.NewHelper(logger).Info("http routes registered")
	return srv
}
