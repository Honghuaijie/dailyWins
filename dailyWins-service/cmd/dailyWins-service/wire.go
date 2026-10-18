//go:build wireinject
// +build wireinject

package main

import (
	"dailyWins-service/internal/auth"
	"dailyWins-service/internal/biz"
	"dailyWins-service/internal/conf"
	"dailyWins-service/internal/data"
	"dailyWins-service/internal/server"
	"dailyWins-service/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		auth.NewOptions,
		server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		newApp,
	))
}
