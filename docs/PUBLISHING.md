# 发布维护说明

源码仓库：https://github.com/hellokitty980808-ctrl/im-core

默认分支为 `main`，项目采用 MIT。第三方许可和 NOTICE 见 `THIRD_PARTY_NOTICES.md` 与 `docs/licenses/`。

## 发布检查

- 运行 `make test`、`make vet`、`make build` 和 `make audit`，按行为改动运行相关真实 E2E。
- 修改协议后运行 `make generate`；检查生成差异和迁移兼容性。
- 检查待提交文件，禁止提交 `.env`、私钥、真实数据、日志、数据卷、工具二进制和本地备份。
- 更新 README、首版或版本说明及验证记录，明确通过与未验证范围。
- 推送后核对远端提交、README、许可证和文件清单。

## 版本与分发

版本标签和 GitHub Release 在确定版本号后单独创建。源码发布不附应用二进制或容器镜像；以后分发这些产物时，应同时提供相应依赖许可、NOTICE 和源码获取说明。
