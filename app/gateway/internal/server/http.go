package server

import (
	"encoding/json"
	"io"
	nethttp "net/http"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/gateway/internal/biz"
	"github.com/hellokitty980808-ctrl/im-core/app/gateway/internal/service"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func requestDecoder(req *nethttp.Request, value any) error {
	message, ok := value.(proto.Message)
	if !ok {
		return errors.BadRequest("INVALID_ARGUMENT", "请求格式错误")
	}
	raw, err := io.ReadAll(io.LimitReader(req.Body, 16385))
	if err != nil {
		return errors.BadRequest("INVALID_ARGUMENT", "请求读取失败")
	}
	if len(raw) > 16384 {
		return errors.BadRequest("INVALID_ARGUMENT", "请求体过大")
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if err = protojson.Unmarshal(raw, message); err != nil {
		return errors.BadRequest("INVALID_ARGUMENT", "JSON 请求字段错误")
	}
	return nil
}
func responseEncoder(w nethttp.ResponseWriter, _ *nethttp.Request, value any) error {
	message, ok := value.(proto.Message)
	if !ok {
		return errors.InternalServer("ENCODING_FAILED", "响应编码失败")
	}
	encoded, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(message)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(encoded)
	return err
}
func errorEncoder(w nethttp.ResponseWriter, _ *nethttp.Request, err error) {
	failure := errors.FromError(err)
	message := failure.Message
	if failure.Code >= 500 {
		message = "服务暂时不可用"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(failure.Code))
	_ = json.NewEncoder(w).Encode(struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{Error: struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    failure.Reason,
		Message: message,
	}})
}

// NewHTTPServer 只注册 v2 公开协议，错误使用 HTTP 状态和结构化 error。
func NewHTTPServer(
	r *platform.Runtime,
	auth *biz.Authenticator,
	accounts *service.AccountProxy,
	conversations *service.ConversationProxy,
) *http.Server {
	srv := http.NewServer(
		http.Address(r.Config.HttpAddr),
		http.Middleware(recovery.Recovery(), ratelimit.Server(), auth.Middleware()),
		http.RequestDecoder(requestDecoder),
		http.ResponseEncoder(responseEncoder),
		http.ErrorEncoder(errorEncoder),
	)
	pb.RegisterAccountServiceHTTPServer(srv, accounts)
	pb.RegisterConversationServiceHTTPServer(srv, conversations)
	srv.HandleFunc("/healthz", func(w nethttp.ResponseWriter, req *nethttp.Request) {
		if req.Method != "GET" && req.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
	})
	return srv
}
