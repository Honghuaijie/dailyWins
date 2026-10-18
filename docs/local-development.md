# 本地启动与数据库准备

## 环境和目录

- 后端：Go 1.22.4+，Kratos + GORM + MySQL。当前验证环境 Go 1.25.1。
- 数据库：MySQL 8，utf8mb4。
- 前端：Node.js 18+，原生 ES Modules；无前端运行时依赖、无需打包。
- 项目：`F:\dev\dailyWins`，后端 `dailyWins-service`，前端 `dailyWins-front`。

## 1. 生成后端代码

现有生成物已在工作区生成。干净检出后，由于模板忽略了 API/Wire 生成物，先安装 protoc 和 Makefile 指定版本的 Go 生成器，再执行：

```powershell
cd F:\dev\dailyWins\dailyWins-service
# 首次缺少生成器时：make init（需要 Go 依赖下载网络）
.\scripts\generate.ps1
go test ./...
go build -o ./bin/dailyWins-service.exe ./cmd/dailyWins-service
```

也可沿用 `make config`、`make api`、`make wire`、`make build`。新增的唯一功能依赖是 `golang.org/x/crypto` 的 bcrypt；原框架和数据库依赖未替换。Go tidy 会补齐已有框架所需的间接依赖。

## 2. 准备数据库与配置

在自己的 MySQL 管理工具中创建专用数据库，例如：

```sql
CREATE DATABASE IF NOT EXISTS daily_wins CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

创建仅能访问该库的账号。由于沿用 AutoMigrate，启动/迁移账号需要该库的建表、索引、必要 ALTER 权限，以及 SELECT/INSERT/UPDATE/DELETE；不要授予其他库的权限。不需要 Redis。

`configs/.env.example` 已提供数据库、服务地址和认证配置说明。**已有 configs/.env 时保留原文件，手动补充缺少项，禁止直接覆盖。** 只有文件不存在时才复制：

```powershell
if (!(Test-Path ./configs/.env)) { Copy-Item ./configs/.env.example ./configs/.env }
```

填写 DB_HOST、DB_PORT、DB_USER、DB_PASSWORD、DB_NAME，并设置：

| 环境变量 | 用途/默认值 |
|---|---|
| HTTP_ADDR | 覆盖 YAML HTTP 地址；本地建议 127.0.0.1:10006 |
| GRPC_ADDR | 覆盖 YAML gRPC 地址；本地建议 127.0.0.1:11006；无业务注册 |
| AUTH_SECRET | 至少 32 字节的随机密钥，无默认值；占位值拒绝启动 |
| AUTH_TTL_HOURS | 默认 168，范围 1–720 |
| COOKIE_SECURE | 程序默认 true；本地 HTTP 显式设 false，生产 HTTPS 为 true |
| ALLOWED_ORIGINS | 默认空，同源代理无需填写；跨域才设明确来源列表 |

可在 PowerShell 临时生成密钥而不写入历史中的明文常量：

```powershell
$bytes = New-Object byte[] 32
$rng = [Security.Cryptography.RandomNumberGenerator]::Create()
$rng.GetBytes($bytes)
$rng.Dispose()
$env:AUTH_SECRET = [Convert]::ToBase64String($bytes)
$env:COOKIE_SECURE = 'false'
$env:HTTP_ADDR = '127.0.0.1:10006'
$env:GRPC_ADDR = '127.0.0.1:11006'
```

临时密钥在终端关闭后消失；正式使用需安全保存到未提交的配置或密钥管理器，重启使用同一个密钥。更换密钥会使所有旧 JWT 失效。操作系统环境变量优先于 configs/.env。不要提交真实账号、密码或密钥。

## 3. 迁移与初始化个人账号

```powershell
go run ./cmd/dailyWins-service -conf ./configs -env local -command migrate .\scripts\init-account.ps1
```

迁移实现是 `internal/data/migrate.go`，沿用 GORM AutoMigrate，正常启动也会调用；可重复执行，不删表、不删历史数据。新增 dw_accounts、dw_sessions、dw_tasks、dw_task_requests；模板 users 表仍保留。新任务日期用 CHAR(10)，时间戳独立存储；基础数据库配置为 UTC。

初始化脚本交互读取账号与隐藏密码，临时传入 INIT_USERNAME/INIT_PASSWORD，结束后清理进程环境变量。密码至少 **6 个字符**，且最多 **72 字节**（中文占多个字节），仅保存 bcrypt cost=12 哈希。固定账号 ID=1，重复初始化直接失败，不覆盖现有账号或密码。也可由安全的进程环境注入 INIT_USERNAME/INIT_PASSWORD 后运行 `-command init-account`。没有注册接口或默认账号密码。

## 4. 启动前后端

终端一：

```powershell
cd F:\dev\dailyWins\dailyWins-service
go run ./cmd/dailyWins-service -conf ./configs -env local
```

终端二：

```powershell
cd F:\dev\dailyWins\dailyWins-front
$env:API_TARGET = 'http://127.0.0.1:10006'
npm start
```

浏览器打开 http://localhost:5173 。前端 Node 服务将 `/v1/*` 转发到固定 API_TARGET，保留 Host/Origin，Cookie 对浏览器同源。可用 PORT 改前端端口；不要直接使用 file:// 打开 HTML。

正式部署时将静态文件和 `/v1/` 放在同一个 HTTPS 站点；设置 COOKIE_SECURE=true，代理终止 TLS 时在 ALLOWED_ORIGINS 配置准确站点来源。此任务没有执行任何公网部署。

## 5. 旧数据与保存行为

正式页面不会生成示例任务。浏览器存在 `daily-wins:v1` 时，显示保留提示与下载备份按钮；不清除、不上传、不自动合并。下方月历只显示云端记录。需要转移的旧任务可参考备份手动补录；V1 不提供批量导入。

新增点击添加建立任务，编辑已有任务 450ms 后自动保存。页面上的即时勾选/达成反馈为本地预览，是否持久化以保存状态为准。网络失败保留当前会话内任务与草稿并支持重试；切换日期不丢输入。关闭/刷新有未保存输入时会提示。草稿只在内存，不承诺浏览器崩溃恢复。版本冲突保留输入，导出备份后重新加载核对，不自动覆盖其他设备的新版本。

## 6. 验证

```powershell
# 后端
go test ./...
go vet ./...
go build -o ./bin/dailyWins-service.exe ./cmd/dailyWins-service
# 前端目录
npm test
npm run check
```

真实 MySQL 测试要求独立库名以 `dailywins_test_` 开头，显式设置 DAILYWINS_TEST_DSN；不要指向个人数据。

```powershell
# 在安全终端设置 DAILYWINS_TEST_DSN，含 parseTime=true&loc=UTC
go test ./internal/data -run TestMySQLPersistence -v -count=1
```

测试会在该独立库重复执行迁移，业务测试数据使用事务回滚；不清空或删除数据库。未设置变量时明确 skip。

无外部数据库的浏览器联调，可启动显式测试夹具（仅内存仓储）：

```powershell
# 后端目录；夹具最多运行20分钟，结束用 Ctrl+C
$env:DAILYWINS_BROWSER_TEST = '1'
go test ./internal/server -run '^TestBrowserHarness$' -count=1 -timeout=25m -v
# 新终端，前端目录
$env:API_TARGET = Get-Content ../.cache/browser-api-url.txt
npm start
# 再开终端，前端目录（安装/配置 Playwright + Edge）
$env:E2E_USERNAME = 'owner'
$env:E2E_PASSWORD = 'test-password-long' # 仅测试夹具内存账号，无正式权限
node tests/browser.cjs
```

浏览器用例要求独立空测试账号，会创建测试记录。夹具重启即可重置内存，不能用它验证数据库持久化。当前执行结果及未验证项见 `test-results.md`。
