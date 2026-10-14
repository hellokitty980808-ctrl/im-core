//go:build wireinject

package main

import (
	"github.com/hellokitty980808-ctrl/im-core/app/user/service/internal/biz"
	"github.com/hellokitty980808-ctrl/im-core/app/user/service/internal/data"
	"github.com/hellokitty980808-ctrl/im-core/app/user/service/internal/server"
	"github.com/hellokitty980808-ctrl/im-core/app/user/service/internal/service"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	kratos "github.com/go-kratos/kratos/v2"
	"github.com/google/wire"
)

func wireApp(r *platform.Runtime) (*kratos.App, func(), error) {
	panic(wire.Build(
		data.NewRepository,
		biz.NewAccountUseCase,
		service.NewAccountService,
		server.NewGRPCServer,
		newApp,
	))
}
