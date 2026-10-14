// generate 从 v2 Proto 和 Wire 源生成服务协议、HTTP、配置和 OpenAPI。
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
func main() {
	root, err := os.Getwd()
	if err == nil {
		err = os.Setenv("PATH", filepath.Join(root, ".tools")+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	if err == nil {
		err = run(
			"protoc",
			"-I.",
			"-Ithird_party",
			"--go_out=paths=source_relative:.",
			"--go-grpc_out=paths=source_relative:.",
			"--go-http_out=paths=source_relative:.",
			"api/im/v2/im.proto",
		)
	}
	if err == nil {
		err = run("protoc", "-I.", "--go_out=paths=source_relative:.", "pkg/platform/config.proto")
	}
	if err == nil {
		err = run(
			"protoc",
			"-I.",
			"-Ithird_party",
			"--openapi_out=paths=source_relative:docs",
			"api/im/v2/im.proto",
		)
	}
	if err == nil {
		err = run(
			"wire",
			"./app/user/service/cmd/server",
			"./app/chat/service/cmd/server",
			"./app/connect/service/cmd/server",
			"./app/gateway/cmd/server",
		)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
