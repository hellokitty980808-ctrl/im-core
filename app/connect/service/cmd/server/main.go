package main

import (
	"context"

	"github.com/hellokitty980808-ctrl/im-core/app/connect/service/internal/biz"
	"github.com/hellokitty980808-ctrl/im-core/pkg/delivery"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	kratos "github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

func newApp(r *platform.Runtime, gs *grpc.Server, hs *http.Server, hub *biz.Hub) *kratos.App {
	worker := platform.NewWorker(func(ctx context.Context) {
		delivery.Subscribe(ctx, r.Config.BrokerUrl, hub.Deliver, hub.SetReady, r.Logger)
	})
	return kratos.New(
		kratos.ID(r.ID),
		kratos.Name(r.Name),
		kratos.Logger(r.Logger),
		kratos.Registrar(r.Registry),
		kratos.Server(gs, hs, worker),
		kratos.BeforeStop(func(context.Context) error {
			hub.Close()
			return nil
		}),
	)
}
func main() {
	runtime, err := platform.New("connect")
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
