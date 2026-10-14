package main

import (
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	kratos "github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/http"
)

func newApp(r *platform.Runtime, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(r.ID),
		kratos.Name(r.Name),
		kratos.Logger(r.Logger),
		kratos.Registrar(r.Registry),
		kratos.Server(hs),
	)
}
func main() {
	runtime, err := platform.New("gateway")
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
