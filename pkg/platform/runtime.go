// Package platform 管理服务配置、注册发现和资源生命周期。
package platform

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	registryconsul "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/uuid"
	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

// Runtime 持有每个进程的配置和日志；敏感配置不会写入日志。
type Runtime struct {
	Name      string
	ID        string
	Config    *Config
	Logger    log.Logger
	Registry  registry.Registrar
	Discovery registry.Discovery
}

// New 从指定配置目录加载配置，基础设施凭据仅来自环境变量。
func New(name string) (*Runtime, error) {
	path := flag.String("conf", "configs", "配置目录")
	flag.Parse()
	raw, err := os.ReadFile(filepath.Join(*path, "config.yaml"))
	if err != nil {
		return nil, err
	}
	var values map[string]any
	if err = yaml.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err = protojson.Unmarshal(encoded, cfg); err != nil {
		return nil, err
	}
	env := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	cfg.MysqlDsn = fmt.Sprintf(
		"%s:%s@tcp(%s:3306)/%s_%s?charset=utf8mb4&parseTime=true&multiStatements=true",
		env("DB_USER", "root"),
		os.Getenv("DB_PASSWORD"),
		env("DB_HOST", "mysql"),
		env("DB_NAME_PREFIX", "imcore"),
		name,
	)
	cfg.RedisAddr = env("REDIS_HOST", "redis") + ":6379"
	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")
	cfg.ConsulAddr = env("CONSUL_ADDR", "consul:8500")
	broker := &url.URL{
		Scheme: "amqp",
		Host:   env("RABBITMQ_HOST", "rabbitmq") + ":5672",
		Path:   "/",
		User:   url.UserPassword(env("RABBITMQ_USER", "imcore"), os.Getenv("RABBITMQ_PASSWORD")),
	}
	cfg.BrokerUrl = broker.String()
	cfg.PrivateKey = os.Getenv("JWT_PRIVATE_KEY")
	cfg.PublicKey = os.Getenv("JWT_PUBLIC_KEY")
	cfg.CaptchaDisabled = os.Getenv("CAPTCHA_DISABLED") == "true"
	client, err := consulapi.NewClient(&consulapi.Config{Address: cfg.ConsulAddr})
	if err != nil {
		return nil, err
	}
	reg := registryconsul.New(client)
	return &Runtime{
		Name:      "imcore." + name,
		ID:        uuid.NewString(),
		Config:    cfg,
		Logger:    log.With(log.NewStdLogger(os.Stdout), "service", name, "ts", log.DefaultTimestamp),
		Registry:  reg,
		Discovery: reg,
	}, nil
}

// Dial 使用注册发现连接内部服务，调用超时为五秒。
func (r *Runtime) Dial(service string) (*grpc.ClientConn, error) {
	return kgrpc.DialInsecure(
		context.Background(),
		kgrpc.WithEndpoint("discovery:///imcore."+service),
		kgrpc.WithDiscovery(r.Discovery),
		kgrpc.WithTimeout(5*time.Second),
	)
}
