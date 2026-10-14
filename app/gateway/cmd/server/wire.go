//go:build wireinject

package main

import (
	"github.com/hellokitty980808-ctrl/im-core/app/gateway/internal/biz"
	"github.com/hellokitty980808-ctrl/im-core/app/gateway/internal/data"
	"github.com/hellokitty980808-ctrl/im-core/app/gateway/internal/server"
	"github.com/hellokitty980808-ctrl/im-core/app/gateway/internal/service"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	kratos "github.com/go-kratos/kratos/v2"
	"github.com/google/wire"
)

func wireApp(r *platform.Runtime) (*kratos.App, func(), error) {
	panic(wire.Build(
		data.NewClients,
		biz.NewAuthenticator,
		service.NewAccountProxy,
		service.NewConversationProxy,
		server.NewHTTPServer,
		newApp,
	))
}
