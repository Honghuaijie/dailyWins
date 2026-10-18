package handler

import (
	stdhttp "net/http"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func HealthCheckHandler(ctx kratoshttp.Context) error {
	return ctx.JSON(stdhttp.StatusOK, map[string]string{
		"status": "ok",
	})
}
