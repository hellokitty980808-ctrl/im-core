package main

import (
	"context"

	"github.com/hellokitty980808-ctrl/im-core/app/user/service/internal/data"
	"github.com/hellokitty980808-ctrl/im-core/pkg/delivery"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	kratos "github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func newApp(r *platform.Runtime, gs *grpc.Server, repo *data.Repository) *kratos.App {
	worker := platform.NewWorker(func(ctx context.Context) {
		delivery.PublishLoop(ctx, repo.DB, r.Config.BrokerUrl, r.Logger)
	})
	return kratos.New(
		kratos.ID(r.ID),
		kratos.Name(r.Name),
		kratos.Logger(r.Logger),
		kratos.Registrar(r.Registry),
		kratos.Server(gs, worker),
	)
}
func main() {
	runtime, err := platform.New("user")
	if err != nil {
		panic(err)
	}
	app, close, err := wireApp(runtime)
	if err != nil {
		panic(err)
	}
	defer close()
	if err = app.Run(); err != nil {
		panic(err)
	}
}
