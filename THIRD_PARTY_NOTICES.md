# 第三方许可与归属

项目自身计划使用 MIT，第三方代码仍遵循各自许可证。根目录 `LICENSE` 不替代依赖的许可文本、版权声明或 NOTICE。

本清单依据当前 `go.mod`、实际构建与测试依赖以及 `make tools` 固定的生成工具版本整理；只复制许可和归属文件，没有将依赖源码 vendoring 到本仓库。工具和运行时使用同一模块的不同版本时分别记录。依赖升级后需要重新核对并更新。

## Go 模块和工具

| 模块 | 版本 | 许可识别 | 使用范围 | 完整原文 |
|---|---|---|---|---|
| `filippo.io/edwards25519` | `v1.2.0` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/filippo.io__edwards25519@v1.2.0/LICENSE) |
| `github.com/armon/go-metrics` | `v0.4.1` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__armon__go-metrics@v0.4.1/LICENSE) |
| `github.com/cespare/xxhash/v2` | `v2.3.0` | MIT | 构建/测试/直接依赖 | [LICENSE.txt](docs/licenses/github.com__cespare__xxhash__v2@v2.3.0/LICENSE.txt) |
| `github.com/fatih/color` | `v1.19.0` | MIT | 构建/测试/直接依赖 | [LICENSE.md](docs/licenses/github.com__fatih__color@v1.19.0/LICENSE.md) |
| `github.com/go-kratos/aegis` | `v0.2.0` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__go-kratos__aegis@v0.2.0/LICENSE) |
| `github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2` | `v2.0.0-20260404020628-f149714c1d54` | MIT | 协议生成工具 | [LICENSE](docs/licenses/github.com__go-kratos__kratos__cmd__protoc-gen-go-http__v2@v2.0.0-20260404020628-f149714c1d54/LICENSE) |
| `github.com/go-kratos/kratos/contrib/registry/consul/v2` | `v2.0.0-20260404020628-f149714c1d54` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__go-kratos__kratos__contrib__registry__consul__v2@v2.0.0-20260404020628-f149714c1d54/LICENSE) |
| `github.com/go-kratos/kratos/v2` | `v2.9.3-0.20251217105121-fb8e43efb207` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__go-kratos__kratos__v2@v2.9.3-0.20251217105121-fb8e43efb207/LICENSE) |
| `github.com/go-playground/form/v4` | `v4.3.0` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__go-playground__form__v4@v4.3.0/LICENSE) |
| `github.com/go-sql-driver/mysql` | `v1.10.0` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__go-sql-driver__mysql@v1.10.0/LICENSE) |
| `github.com/go-viper/mapstructure/v2` | `v2.5.0` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__go-viper__mapstructure__v2@v2.5.0/LICENSE) |
| `github.com/golang-jwt/jwt/v5` | `v5.3.1` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__golang-jwt__jwt__v5@v5.3.1/LICENSE) |
| `github.com/golang/freetype` | `v0.0.0-20170609003504-e2365dfdc4a0` | FTL（上游另提供 GPL-2.0-or-later 选项） | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__golang__freetype@v0.0.0-20170609003504-e2365dfdc4a0/LICENSE)、[FTL](docs/licenses/github.com__golang__freetype@v0.0.0-20170609003504-e2365dfdc4a0/ftl.txt)、[GPL](docs/licenses/github.com__golang__freetype@v0.0.0-20170609003504-e2365dfdc4a0/gpl.txt) |
| `github.com/google/gnostic` | `v0.7.0` | Apache-2.0 | 协议生成工具 | [LICENSE](docs/licenses/github.com__google__gnostic@v0.7.0/LICENSE) |
| `github.com/google/uuid` | `v1.6.0` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__google__uuid@v1.6.0/LICENSE) |
| `github.com/google/wire` | `v0.7.0` | Apache-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__google__wire@v0.7.0/LICENSE) |
| `github.com/gorilla/mux` | `v1.8.1` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__gorilla__mux@v1.8.1/LICENSE) |
| `github.com/gorilla/websocket` | `v1.5.4-0.20250319132907-e064f32e3674` | BSD-2-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__gorilla__websocket@v1.5.4-0.20250319132907-e064f32e3674/LICENSE) |
| `github.com/hashicorp/consul/api` | `v1.34.4` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__consul__api@v1.34.4/LICENSE) |
| `github.com/hashicorp/errwrap` | `v1.1.0` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__errwrap@v1.1.0/LICENSE) |
| `github.com/hashicorp/go-cleanhttp` | `v0.5.2` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__go-cleanhttp@v0.5.2/LICENSE) |
| `github.com/hashicorp/go-hclog` | `v1.6.3` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__go-hclog@v1.6.3/LICENSE) |
| `github.com/hashicorp/go-immutable-radix` | `v1.3.1` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__go-immutable-radix@v1.3.1/LICENSE) |
| `github.com/hashicorp/go-metrics` | `v0.6.0` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__go-metrics@v0.6.0/LICENSE) |
| `github.com/hashicorp/go-multierror` | `v1.1.1` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__go-multierror@v1.1.1/LICENSE) |
| `github.com/hashicorp/go-rootcerts` | `v1.0.2` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__go-rootcerts@v1.0.2/LICENSE) |
| `github.com/hashicorp/golang-lru` | `v1.0.2` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__golang-lru@v1.0.2/LICENSE) |
| `github.com/hashicorp/serf` | `v0.10.4` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__hashicorp__serf@v0.10.4/LICENSE) |
| `github.com/jinzhu/inflection` | `v1.0.0` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__jinzhu__inflection@v1.0.0/LICENSE) |
| `github.com/jinzhu/now` | `v1.1.5` | MIT | 构建/测试/直接依赖 | [License](docs/licenses/github.com__jinzhu__now@v1.1.5/License) |
| `github.com/mattn/go-colorable` | `v0.1.15` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__mattn__go-colorable@v0.1.15/LICENSE) |
| `github.com/mattn/go-isatty` | `v0.0.22` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__mattn__go-isatty@v0.0.22/LICENSE) |
| `github.com/mitchellh/go-homedir` | `v1.1.0` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__mitchellh__go-homedir@v1.1.0/LICENSE) |
| `github.com/mojocn/base64Captcha` | `v1.3.8` | Apache-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__mojocn__base64Captcha@v1.3.8/LICENSE) |
| `github.com/rabbitmq/amqp091-go` | `v1.12.0` | BSD-2-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__rabbitmq__amqp091-go@v1.12.0/LICENSE) |
| `github.com/redis/go-redis/v9` | `v9.21.0` | BSD-2-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__redis__go-redis__v9@v9.21.0/LICENSE) |
| `github.com/shirou/gopsutil/v3` | `v3.23.6` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__shirou__gopsutil__v3@v3.23.6/LICENSE) |
| `github.com/shoenig/go-m1cpu` | `v0.1.6` | MPL-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__shoenig__go-m1cpu@v0.1.6/LICENSE) |
| `github.com/tklauser/go-sysconf` | `v0.3.11` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/github.com__tklauser__go-sysconf@v0.3.11/LICENSE) |
| `go.uber.org/atomic` | `v1.11.0` | MIT | 构建/测试/直接依赖 | [LICENSE.txt](docs/licenses/go.uber.org__atomic@v1.11.0/LICENSE.txt) |
| `golang.org/x/crypto` | `v0.54.0` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/golang.org__x__crypto@v0.54.0/LICENSE) |
| `golang.org/x/exp` | `v0.0.0-20260603202125-055de637280b` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/golang.org__x__exp@v0.0.0-20260603202125-055de637280b/LICENSE) |
| `golang.org/x/image` | `v0.38.0` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/golang.org__x__image@v0.38.0/LICENSE) |
| `golang.org/x/net` | `v0.57.0` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/golang.org__x__net@v0.57.0/LICENSE) |
| `golang.org/x/sync` | `v0.22.0` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/golang.org__x__sync@v0.22.0/LICENSE) |
| `golang.org/x/sys` | `v0.47.0` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/golang.org__x__sys@v0.47.0/LICENSE) |
| `golang.org/x/text` | `v0.40.0` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/golang.org__x__text@v0.40.0/LICENSE) |
| `google.golang.org/genproto/googleapis/api` | `v0.0.0-20260715232425-e75dac1f907d` | Apache-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/google.golang.org__genproto__googleapis__api@v0.0.0-20260715232425-e75dac1f907d/LICENSE) |
| `google.golang.org/genproto/googleapis/rpc` | `v0.0.0-20260706201446-f0a921348800` | Apache-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/google.golang.org__genproto__googleapis__rpc@v0.0.0-20260706201446-f0a921348800/LICENSE) |
| `google.golang.org/grpc` | `v1.82.1` | Apache-2.0 | 构建/测试/直接依赖 | [LICENSE](docs/licenses/google.golang.org__grpc@v1.82.1/LICENSE), [NOTICE.txt](docs/licenses/google.golang.org__grpc@v1.82.1/NOTICE.txt) |
| `google.golang.org/grpc/cmd/protoc-gen-go-grpc` | `v1.6.2` | Apache-2.0 | 协议生成工具 | [LICENSE](docs/licenses/google.golang.org__grpc__cmd__protoc-gen-go-grpc@v1.6.2/LICENSE) |
| `google.golang.org/protobuf` | `v1.36.11` | BSD-3-Clause | 协议生成工具 | [LICENSE](docs/licenses/google.golang.org__protobuf@v1.36.11/LICENSE) |
| `google.golang.org/protobuf` | `v1.36.12` | BSD-3-Clause | 构建/测试/直接依赖 | [LICENSE](docs/licenses/google.golang.org__protobuf@v1.36.12/LICENSE) |
| `gopkg.in/yaml.v3` | `v3.0.1` | MIT / Apache-2.0（按文件归属） | 构建/测试/直接依赖 | [LICENSE](docs/licenses/gopkg.in__yaml.v3@v3.0.1/LICENSE), [NOTICE](docs/licenses/gopkg.in__yaml.v3@v3.0.1/NOTICE) |
| `gorm.io/driver/mysql` | `v1.6.0` | MIT | 构建/测试/直接依赖 | [License](docs/licenses/gorm.io__driver__mysql@v1.6.0/License) |
| `gorm.io/gorm` | `v1.31.2` | MIT | 构建/测试/直接依赖 | [LICENSE](docs/licenses/gorm.io__gorm@v1.31.2/LICENSE) |

许可证识别只用于导航；具体授权条件以链接中的原文及各源文件声明为准。BSD 条款数按当前根许可文本识别，模块内部可能包含另有声明的文件。

## FreeType-Go 归属

本项目的依赖包含 FreeType-Go 的文字栅格化代码。FreeType-Go 提供 FTL 或 GPL-2.0-or-later 两种许可选择；本项目按 FTL 分支整理使用和归属说明。其源码声明为：

> Copyright 2010 The Freetype-Go Authors. All rights reserved.

FreeType 项目及作者的版权声明和 FTL 条款完整保留在对应附件；许可文字由上游编码转换为 UTF-8，内容未改写。GPL 文本作为上游备选许可一并保留，不表示项目自身改用 GPL。上游测试字体有单独声明，本仓库未复制这些测试字体。

## 随仓库分发的标准协议

| 文件 | 归属 | 许可证 |
|---|---|---|
| `third_party/google/api/annotations.proto` | Google | Apache-2.0；原文件头声明保留 |
| `third_party/google/api/http.proto` | Google | Apache-2.0；原文件头声明保留 |
| `third_party/google/protobuf/descriptor.proto` | Google / Protocol Buffers | BSD-3-Clause；原文件头及条款保留 |
| `third_party/google/protobuf/empty.proto` | Google / Protocol Buffers | BSD-3-Clause；原文件头及条款保留 |

Apache-2.0 完整文本随上述模块许可附件提供；标准协议的原版权和许可声明仍位于各文件头部。

## 分发范围

本次准备发布源码，不包含编译二进制、容器镜像或基础设施镜像。Docker Compose 引用的 MySQL、Valkey、RabbitMQ、Consul 等镜像由各自上游分发；这里没有重新许可这些镜像。

MySQL 驱动、Consul API 和部分间接依赖使用 MPL-2.0，原许可保持有效。以后如果分发应用二进制或容器镜像，需要同时提供适用依赖的许可、NOTICE 和源码获取说明，不能只附项目的 MIT 文本。参见 [MPL-2.0 原文](https://www.mozilla.org/en-US/MPL/2.0/)。
