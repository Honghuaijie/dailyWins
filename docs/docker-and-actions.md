# Docker 镜像与 GitHub Actions

## 自动发布

根仓库 `.github/workflows/` 中有两条独立工作流，参考 lifevault 的 GHCR 发布方式：

| master 上的变更 | 发布镜像 |
| --- | --- |
| `dailyWins-front/**` 或前端 workflow | `ghcr.io/<小写 GitHub 用户或组织名>/dailywins-front` |
| `dailyWins-service/**` 或后端 workflow | `ghcr.io/<小写 GitHub 用户或组织名>/dailywins-service` |
| 两端都有改动 | 两个镜像分别构建、测试、发布 |
| 只有根目录 docs 等其他文件 | 不触发 |

支持 Actions 页面手动 Run workflow，仅 master 可以发布。标签包括 `latest`、`master-<短 SHA>`、`sha-<短 SHA>`；发布完成后 Summary 显示完整镜像名、可复制的 `./deploy.sh <镜像名>` 命令和 digest。部署脚本需预先放在服务器对应的项目部署目录中；工作流只发布镜像并展示命令，不会远程连接或自动部署。两条工作流使用独立构建缓存和并发组，同一端发布串行运行。

前端无需打包工具：镜像构建先执行语法检查和 Node 测试，再复制网页与同源代理到运行镜像。后端在镜像内安装现有 Makefile 指定的生成器，生成 protobuf/Wire 代码，执行 `go test ./...`，再编译 Linux 二进制；不依赖本机生成产物。Go 构建镜像使用 1.25，与本地验证版本一致，不修改 go.mod。

## GitHub 配置

1. 将整个 dailyWins 根仓库提交并推送到 GitHub 的 `master`，包括根目录 `.github` 和两端源代码。
2. 确认仓库 Actions 已启用，允许使用工作流引用的 actions/docker Actions。组织策略需允许 `packages: write`。
3. 无需新建 PAT、Docker Hub 密码或业务 Secrets。工作流使用 GitHub 自动提供的 `GITHUB_TOKEN` 登录 GHCR，权限限定为 `contents: read` 和发布任务的 `packages: write`。数据库、账号密码、AUTH_SECRET 都不需要放进 CI。
4. 如果同名 GHCR Package 已存在，在该 Package 的 Settings → Manage Actions access 授权本仓库写入；否则首次发布自动关联仓库。
5. GitHub 仓库 public 不代表镜像自动 public。需要匿名拉取时，在首次发布后的 Package Settings 中确认/调整镜像可见性为 Public。

参考：[GitHub 发布 Docker 镜像](https://docs.github.com/en/actions/tutorials/publish-packages/publish-docker-images)、[GHCR 权限与可见性](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)。

## 本地构建

在 dailyWins 根目录执行，需要 Docker 引擎运行并切换为 Linux containers：

```powershell
docker build -t dailywins-service:local ./dailyWins-service
docker build -t dailywins-front:local ./dailyWins-front
```

每端使用自己的目录作为 build context。`.dockerignore` 排除真实 `.env`、密钥、本地生成文件和日志；后端运行镜像只复制二进制与 `config.yaml`、`config-prod.yaml`。前后端均使用非 root 用户运行。

## 本地容器联调

先按 `local-development.md` 准备 MySQL 和后端 `configs/.env`，不要提交真实配置。下面命令使用 Docker Desktop 连接宿主机 MySQL（MySQL 需允许该连接）；若 MySQL 在其他机器或容器中，将 DB_HOST 换为可达主机名。

```powershell
docker network create dailywins
docker run -d --name backend --network dailywins --env-file ./dailyWins-service/configs/.env -e DB_HOST=host.docker.internal -e HTTP_ADDR=0.0.0.0:10006 -e GRPC_ADDR=0.0.0.0:11006 -e COOKIE_SECURE=false dailywins-service:local
docker run -d --name frontend --network dailywins -p 127.0.0.1:5173:5173 -e API_TARGET=http://backend:10006 dailywins-front:local
```

访问 `http://localhost:5173`。后端端口仅供容器网络访问，前端代理 `/v1/*`，浏览器仍是同源访问。已有账号和任务保留在原数据库。后端启动沿用现有自动迁移机制；也可单独执行：

```powershell
docker run --rm --network dailywins --env-file ./dailyWins-service/configs/.env -e DB_HOST=host.docker.internal dailywins-service:local -conf /data/conf -env prod -command migrate
```

账号初始化仍可使用原来的 `dailyWins-service/scripts/init-account.ps1`，连接相同数据库即可，不需要在 Dockerfile 或 workflow 中写密码。

正式 HTTPS 反向代理环境必须设置 `COOKIE_SECURE=true`，并将 `ALLOWED_ORIGINS` 设置为实际外部来源（例如 `https://wins.example.com`，不带路径），因为 TLS 终止后后端收到的是 HTTP。`API_TARGET` 是运行时环境变量；容器内默认 `http://backend:10006`，不需要重新构建前端。日志默认 `/logs`，如挂载宿主机目录，需允许容器 UID 10001 写入。

## 验证范围

本次本地前端 12 项测试与语法检查通过，后端 `go test ./...` 和 Linux amd64 静态交叉编译通过。两份 workflow YAML 解析通过；使用临时 HTTP 后端验证了前端 HOST 配置、静态页面、同源 API 代理和敏感路径 404。真实 MySQL 集成测试仍需显式测试数据库环境，本次未访问真实数据库。Docker 引擎当前未运行，因此尚未完成实际镜像构建、容器联调或 GHCR 推送；首次推送 master 后请在 Actions 查看两条构建结果。没有将代码推送或部署到公网。
