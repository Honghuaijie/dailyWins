# 项目说明

Daily Wins V1 已实现固定账号登录、可撤销会话和任务 API。完整交付文档位于上级 `docs/`：

- `docs/local-development.md`：数据库、环境变量、账号初始化、前后端启动。
- `docs/api.md`：接口契约与认证、幂等、并发规则。
- `docs/test-results.md`：已验证范围与真实 MySQL 待验证事项。

Windows 可使用 `scripts/generate.ps1` 生成代码、`scripts/init-account.ps1` 隐藏输入密码初始化账号。原模板用户 CRUD 只保留示例源码，正式 HTTP/gRPC 服务不注册。正常启动前需配置数据库与 AUTH_SECRET；不含默认账号密码。

## 目录说明

- `api/<domain>/vN/*.proto`：proto 源文件
- `api/all-pb-go/vN/*.pb.go`：proto 生成物
- `cmd/dailyWins-service`：服务启动入口
- `configs/`：基础配置和环境配置
- `internal/biz`：业务用例和 repo 接口
- `internal/data`：DB、事务和 repo 实现
- `internal/service`：gRPC/HTTP service 实现
- `internal/errors`：统一错误编码
- `third_party/`：常用 proto 依赖

## 开发流程

### 初始化工具

```bash
make init
```

### 生成代码

```bash
make config
make api
make wire
```

### 本地运行

```bash
make run env=local
```

### 运行测试

```bash
make test
```

如果只想运行 `biz` 层测试：

```bash
make test-biz
```

更详细的测试说明见根目录下的 `TESTING_GUIDE.md`。

AI 生成代码时的强约束规范见根目录下的 `AI_CODING_RULES.md`。

## GitHub Actions 镜像构建

模板生成的新项目默认内置后端镜像构建工作流。

- 工作流文件会生成到 `.github/workflows/build-backend-image.yml`
- 推送到 `master` 分支后，会自动构建当前项目根目录的 `Dockerfile`
- 镜像会推送到：

```text
ghcr.io/<github_owner>/dailyWins-service
```

- 标签规则默认包括：
  - `latest`
  - `master-<short-sha>`

工作流 Summary 会直接输出：

- 完整镜像名
- 可复制部署命令，例如：

```bash
./deploy.sh ghcr.io/<github_owner>/dailyWins-service:master-xxxxxxx
```

使用前需要在 GitHub 仓库中确认：

- `Settings -> Actions -> General -> Workflow permissions`
- 选择 `Read and write permissions`

支持环境：

- `local`
- `dev`
- `test`
- `prod`

## 配置加载链路

- 项目生成脚本会创建本地 `configs/.env`，数据库参数写入该文件
- `configs/.env` 被 Git 和 Docker 构建上下文同时忽略，不会进入仓库或镜像
- `configs/.env.example` 是可提交的变量清单，不得包含有效密码或密钥
- 本地启动时会先读取 `configs/.env`；系统中已有的环境变量优先，不会被文件覆盖
- 启动默认读取：`configs/config.yaml`
- 传入 `-env` 时叠加读取：`configs/config-{env}.yaml`
- 环境配置覆盖基础配置的同名字段
- 生产环境不依赖镜像中的 `.env`，由服务器 Docker Compose 的 `env_file` 注入变量

默认示例：

```bash
make run env=local
make run env=dev
```

## API 生成规则

- proto 源目录：`api/<domain>/vN/*.proto`
- 生成物目录：`api/all-pb-go/vN`
- `go_package` 必须与生成目录一致

例如：

```proto
option go_package = "dailyWins-service/api/all-pb-go/v1;v1";
```

如果后续新增 `api/user/v2/user.proto`，生成物必须落到 `api/all-pb-go/v2`，项目内部也应从 `api/all-pb-go/v2` 引用。

## 示例接口

- `POST /v1/users/create`
- `GET /v1/users/list`
- `GET /v1/users/get/{id}`
- `POST /v1/users/update`
- `POST /v1/users/delete`
- `GET /health`
- `GET /healthz`

## 模板默认能力

- `main.go` 直接创建 logger，不使用 `logger_provider.go`
- 启动时默认设置 `Asia/Shanghai`
- HTTP 默认启用 `Recovery + Validate + CORS + RequestLog`
- gRPC 默认启用 `Recovery + RequestLog`
- 错误响应统一输出 `code / message / data`
- 成功响应示例统一输出 `code / message / data`，成功时 `code = 0`
- 示例中请求参数校验放在 `service` 层，`biz` 层专注业务流程
- 默认内置 GHCR 镜像构建 workflow，模板仓库本身不会执行该 workflow

## 错误码规范

- 统一错误码定义集中在 `internal/errors/codes.go`
- 默认错误文案统一使用中文
- 业务层优先通过 `errors.E(...)`、`errors.EWithMessage(...)` 返回错误
- 未显式包装的普通错误会统一兜底为 `50000 / 服务内部错误`
- 日志中统一记录业务错误码 `code` 和 HTTP 状态码 `http_code`

常用示例：

```go
return nil, errors.E(errors.ErrInvalidID)
return nil, errors.EWithMessage(errors.ErrMissingRequiredField, "name 和 email 不能为空")
```

扩展规则：

- `40000-49999`：请求侧和业务侧错误
- `50000-59999`：服务端、依赖、系统错误
- 新项目新增业务域错误时，继续在 `internal/errors/codes.go` 中按号段追加
