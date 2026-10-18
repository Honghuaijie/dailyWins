# AI 后端代码生成规范

## 适用范围

这份文档用于约束 AI 在当前模板体系内生成后端代码时的行为，目标是生成可维护、可测试、分层清晰的后端代码。

适用范围：

- 后端：`Kratos + GORM + MySQL`
- 使用场景：新增业务模块、普通接口、文件上传接口

不适用范围：

- 历史脏代码迁移
- 跨技术栈重构
- 脱离当前模板目录结构的自由发挥

## 全局强制原则

1. 必须先定义接口契约，再写实现。
2. 普通后端接口必须 `proto` 优先。
3. 文件上传接口允许走 HTTP 例外通道。
4. 必须严格遵守分层边界，禁止跨层写逻辑。
5. 必须复用统一错误码、目录结构、命名模式。
6. 必须生成可测试、可 review、可维护的代码。
7. 新增依赖后必须同步更新依赖注入定义，保证项目可编译。
8. 新增接口必须先判断接口安全分类，再决定路由前缀、鉴权中间件和权限校验逻辑。

## 后端编码规范

### 技术栈与目录

后端固定使用：

- `kratos`
- `gorm`
- `mysql`

必须遵守以下目录职责：

- `api/`：`proto` 接口定义
- `internal/service/`：接口实现、参数校验、响应组装
- `internal/biz/`：业务规则、事务编排、领域逻辑
- `internal/data/`：数据库访问、repo 实现、model 转换
- `internal/auth/`：JWT 生成、解析和认证辅助
- `internal/errors/`：统一错误码和错误包装
- `internal/server/http.go`：HTTP 服务注册和文件上传类例外路由
- `cmd/<service>/wire.go`：依赖注入定义

后端代码生成时，AI 必须以当前模板中的 `user` 示例为直接参照物，优先对齐以下文件的写法：

- `api/user/v1/user.proto`
- `internal/service/user.go`
- `internal/biz/user.go`
- `internal/data/user.go`
- `internal/auth/jwt.go`
- `internal/errors/codes.go`
- `internal/server/http.go`

新生成的代码分层、错误处理、响应结构和依赖注入风格必须以示例为准，不允许自行发明另一套规范。接口路径和鉴权规则必须以本文的“接口安全与权限规范”为准；如果现有示例路径与安全分区路径冲突，以安全分区路径为准。

### 后端注释规范

后端核心代码必须补充必要的中文注释，目的是降低维护和交接成本。注释只解释关键意图，不解释代码字面含义。

必须加中文注释的典型位置：

- 事务边界和事务内关键步骤
- 复杂业务分支和状态流转
- 关键字段转换、类型转换和对象组装
- 文件上传、鉴权、幂等、去重等容易误解的逻辑
- 手工注册 HTTP 路由、特殊中间件或例外链路

禁止：

- 每一行都写机械式注释
- 写与代码字面含义重复的废话注释
- 用英文注释替代中文说明
- 只给函数名写一句无信息量的注释，却不解释关键流程

### 接口契约规则

#### 接口安全与权限规范

后端接口必须先按安全边界分成以下三类，再编写 `proto`、`service`、`biz` 和 `data` 实现：

- `user/public`：面向游客和普通用户的公开接口，不需要登录。
- `user/private`：面向普通登录用户的接口，必须用户登录后才能访问。
- `admin`：面向后台管理员的接口，必须管理员登录后才能访问。

接口安全分类必须体现在路由路径上：

- `user/public` 接口使用 `/v1/public/模块名/方法名`
- `user/private` 接口使用 `/v1/user/模块名/方法名`
- `admin` 接口使用 `/v1/admin/模块名/方法名`

如果项目没有普通用户体系，可以没有 `user/private`；如果项目没有公开前台，可以没有 `user/public`。但所有新增接口都必须明确归属到某一类，不允许生成安全边界不清晰的接口。

##### JWT 认证规范

当前模板默认使用 JWT 作为 `user/private` 和 `admin` 接口的认证凭证，认证代码必须优先参考 `internal/auth/jwt.go`。

必须遵守：

- JWT 只能证明“当前请求是谁发起的”，不能替代权限校验和资源归属校验。
- JWT 签名密钥必须只存在于后端，禁止写入前端、H5、小程序、App 或公开仓库。
- JWT 签名密钥必须来自配置或环境变量，生产环境禁止使用默认密钥、空密钥、短密钥或示例密钥。
- 模板默认 claims 只保留 `user_id` 和 `exp`；具体项目可以扩展管理员、角色或权限点，但禁止放入密码、验证码、第三方 secret、数据库连接信息或大量业务数据。
- 生成 JWT 时必须包含过期时间 `exp`；解析 JWT 时必须校验签名算法、签名值和过期时间。
- token 过期、签名错误、缺少 token 时，必须使用统一认证错误码，例如 `ErrMissingToken`、`ErrInvalidToken`、`ErrExpiredToken`、`ErrUnauthorized`。
- `user/private` 接口必须解析出当前用户；`admin` 接口必须解析出当前账号后再判断管理员身份。如果项目没有单独的管理员 ID 字段，可以使用 `user_id` 查询账号并判断是否为管理员。

JWT 校验逻辑必须收敛在统一 helper 或中间件中，禁止每个接口重复手写解析逻辑。推荐命名：

```go
requireUser()
requireAdmin()
requirePermission("article:delete")
```

##### user/public 接口

`user/public` 接口允许任何人访问，目标是提供公开数据，但不能泄露敏感信息，也不能被无限刷。

必须遵守：

- 不校验登录态，只返回公开数据。
- 禁止返回手机号、邮箱、后台备注、草稿内容、内部状态、用户隐私字段、内部配置密钥等敏感数据。
- 必须做参数校验；必须预留或说明限流方案，限流优先放在 Nginx / Kong / API Gateway 等网关层。
- 高频读取接口必要时做缓存。

典型接口：

- `GET /v1/public/articles/list`
- `GET /v1/public/articles/get/{id}`
- `GET /v1/public/tags/list`
- `GET /v1/public/home/get`

##### user/private 接口

`user/private` 接口只允许登录用户访问，并且用户只能访问和操作自己的数据。

必须遵守：

- 必须校验用户登录态，并从 `token`、`session` 或 Cookie 中解析出 `current_user`。
- 查询或修改数据时必须校验资源归属，不能只按资源 ID 操作数据。
- 必须做参数校验和限流；重要操作必须记录操作日志。
- 修改、支付、提现、删除等敏感操作必须有更严格的校验策略。

典型接口：

- `GET /v1/user/me/get`
- `GET /v1/user/orders/list`
- `GET /v1/user/orders/get/{id}`
- `POST /v1/user/profile/update`
- `POST /v1/user/address/delete`

资源归属校验是 `user/private` 接口的强制要求。后端必须先认证“你是谁”，再授权“这个资源是不是你的”。

查询或修改数据时，必须把当前用户 ID 放进条件：

```sql
SELECT * FROM orders
WHERE id = ?
AND user_id = ?;

UPDATE addresses
SET detail = ?
WHERE id = ?
AND user_id = ?;
```

如果查询不到数据或影响行数为 `0`，说明资源不存在或不属于当前用户，应返回 `404` 或 `403`，不得继续执行业务操作。

##### admin 接口

`admin` 接口只允许后台管理员访问，目标是保证只有具备后台身份和对应权限的人才能执行管理操作。

必须遵守：

- 必须校验管理员登录态，并确认当前账号具备后台身份。
- 如果后台存在多个角色，必须校验角色或权限点。
- 必须做参数校验和更严格的限流，登录接口尤其需要限流。
- 重要操作必须记录操作日志；高风险操作可以增加二次确认或二次验证。

典型接口：

- `GET /v1/admin/articles/list`
- `POST /v1/admin/articles/create`
- `POST /v1/admin/articles/delete`
- `POST /v1/admin/site/update`
- `POST /v1/admin/users/disable`

最简单版本允许只校验管理员登录态；多角色后台必须同时校验管理员身份和权限点：

```go
requireAdmin()
requirePermission("article:delete")
```

##### 统一实现结构

新增接口必须遵守以下路由和鉴权边界：

```text
/v1/public/*
    不需要登录

/v1/user/*
    requireUser
    必要时校验资源归属

/v1/admin/*
    requireAdmin
    requirePermission 可选
```

前端隐藏按钮或页面跳转不能作为权限控制依据。所有敏感数据和敏感操作必须以后端校验结果为准。

#### 普通接口

普通接口必须遵守：

- 必须先写 `proto`
- 必须在 `proto` 中定义 `service`、`rpc`、HTTP 路径、请求结构、响应结构
- 必须以 `proto` 作为前后端唯一契约源
- `go_package` 必须符合生成目录约定
- 字段名必须表达明确业务含义，禁止 `data`、`info`、`obj` 这类模糊命名
- 响应结构必须面向业务 DTO，禁止直接暴露数据库模型语义

当前模板强制采用的方法级接口风格：

- 接口路径必须先体现安全分类，再使用方法级接口风格：`/v1/public/模块名/方法名`、`/v1/user/模块名/方法名`、`/v1/admin/模块名/方法名`
- 只允许使用 `GET` 和 `POST`
- 查询类接口优先使用 `GET`
- 创建、更新、删除类接口优先使用 `POST`

以公开文章和后台文章模块为例，正确写法是：

- `GET /v1/public/articles/list`
- `GET /v1/public/articles/get/{id}`
- `GET /v1/admin/articles/list`
- `POST /v1/admin/articles/create`
- `POST /v1/admin/articles/update`
- `POST /v1/admin/articles/delete`

禁止生成 REST 风格的路径，例如：

- `/v1/users`
- `/v1/users/{id}`
- `/v1/users/detail/{id}`

#### 文件上传接口例外

如果接口需要接收文件，允许不走 `proto` 自动注册链路，但必须遵守以下规则：

- 必须在 `internal/server/http.go` 中手工注册 HTTP 路由
- 路径风格仍然必须先体现安全分类，再使用方法级接口风格：`/v1/public/模块名/方法名`、`/v1/user/模块名/方法名`、`/v1/admin/模块名/方法名`
- 上传接口默认使用 `POST`
- 必须在 `service` 中使用 `github.com/go-kratos/kratos/v2/transport/http`
- 必须显式解析 `multipart/form-data`，并校验文本字段、文件字段、大小、类型和数量
- 必须在 `service` 中把文件字节流、文件名、扩展名、MIME 信息整理成 `biz` 可消费的入参，并将 `ctx.Request().Context()` 传给 `biz`
- 必须复用统一错误体系，禁止在上传接口里手写另一套错误 JSON
- 成功响应仍然必须保持 `code / message / data`
- 禁止在 `http.go` 中写业务逻辑，禁止在 `service` 中直接写存储层代码

这类接口是唯一允许绕过 `proto-first` 的推荐例外场景。

正确示例：

```go
srv.Route("").POST("/v1/user/profile/avatarUpload", userSvc.AvatarUpload)
```

### Proto 规范

普通接口的 `proto` 必须满足：

- `request` 和 `reply` 命名必须与接口方法名一一对应
- 一个接口只表达一个明确动作
- 禁止使用不明确宽度的整型，必须使用明确宽度的类型
- 当前模板默认所有业务整型字段优先使用 `int32`
- 不得复用多个接口的 reply 结构
- 每个 reply 都必须包含 `code`、`message`、`data`
- 成功响应时 `code` 固定为 `0`

`request` / `reply` 必须一一对应，例如 `CreateUserRequest` 对应 `CreateUserReply`。禁止复用 `UserReply`、`CommonReply`、`BaseReply` 这类通用 reply。每个接口必须生成独立 reply：

```proto
message CreateUserReply {
  int32 code = 1;
  string message = 2;
  message Data {
    UserInfo user = 1;
  }
  Data data = 3;
}
```

补充要求：

- reply 中的 `Data` 必须定义为该 reply 的内部 message
- 即使两个接口的 `data` 结构相似，也不能直接复用同一个 reply
- 当前模板示例列表接口使用 `items` 和 `total`
- 如果用户没有明确要求分页，不要擅自增加 `page/page_size`

### Service 层规范

`service` 只做三件事：

- 校验参数合法性
- 调用 `biz`
- 构建响应结构

普通接口允许：

- 校验必填、长度、格式、枚举值、分页参数
- 将 `proto request` 转换成 `biz` 入参，并将 `biz` 返回的 `Reply.Data` 包装成最终 `proto reply`

文件上传接口额外允许：

- 使用 `transport/http` 读取 `FormValue`、`FormFile`，校验文件并转换成 `biz` 入参对象

`service` 禁止：

- 直接操作数据库
- 写 SQL / GORM
- 写事务
- 编排复杂业务规则
- 直接依赖 `data`
- 返回存储模型

补充要求：

- `service` 可以接收并向下传递 `context.Context`
- `service` 可以从传输层提取必要元数据并写入上下文；只有 `service` 可以解析 HTTP Header、Cookie、表单和上传文件
- `service` 中涉及业务整型时，必须优先使用 `int32`
- `service` 方法名必须和 `biz` 方法名完全一致
- `service` 成功返回时，`message` 必须使用：`方法名 + " success"`
- 文件上传接口中，`service` 必须先完成参数整理，再调用 `biz`；禁止把 `http.Context` 直接传给 `biz`
- 文件上传接口禁止直接 `ctx.JSON(...map[string]interface{}{})` 手写一套临时响应

例如：

- `CreateUser` 返回 `message: "CreateUser success"`
- `ListUsers` 返回 `message: "ListUsers success"`

具体写法以 `internal/service/user.go` 和 `internal/server/http.go` 中的模板示例为准。

### Biz 层规范

`biz` 是纯业务规则层，允许：

- 接收 `ctx context.Context` 并向下传递
- 业务规则判断
- 多 repo 编排
- 事务控制
- 状态流转、幂等、去重、流程控制
- 文件上传后的业务处理编排

`biz` 禁止：

- 感知 `proto request`
- 感知 Kratos HTTP context
- 解析 HTTP Header 或 Cookie
- 直接解析 `multipart` 请求
- 写 SQL / GORM
- 出现 `*gorm.DB`

文件上传场景下，`biz` 接收的应当是已经整理好的业务参数，例如：

- 文件字节、文件名、扩展名、上传人 ID、业务 ID、其他表单字段

禁止把 `http.Context` 或原始 `multipart` 请求对象透传给 `biz`。上传输入建议采用显式结构体。

事务规则：

- `biz` 层只能通过 `data` 提供的 `Transaction` 接口组织事务
- 禁止在 `biz` 层直接持有或操作 `*gorm.DB`
- 事务范围由业务流程决定，事务实现细节由 `data` 层负责

当前模板的 `biz` 返回规范必须与示例保持一致：

- `biz` 不接收 `proto request`
- `biz` 接收自己的输入对象，例如：
  - `CreateUserInput`
  - `UpdateUserInput`
- `biz` 返回值必须是对应接口的 `Reply.Data` 类型，不返回完整 reply

例如：

```go
func (uc *UserUsecase) CreateUser(ctx context.Context, input *CreateUserInput) (*v1.CreateUserReply_Data, error)
func (uc *UserUsecase) ListUsers(ctx context.Context) (*v1.ListUsersReply_Data, error)
```

方法命名强制要求：

- `service` 叫 `CreateUser`，`biz` 也必须叫 `CreateUser`
- `service` 叫 `ListUsers`，`biz` 也必须叫 `ListUsers`
- 禁止生成 `service.CreateUser` 调 `biz.Create` 这类命名错位

### Data 层规范

`data` 允许：

- 接收 `ctx context.Context`
- CRUD
- 查询条件拼装
- 分页查询
- model 与 domain 对象转换
- 事务内 DB 选择

`data` 禁止：

- 业务规则判断
- 直接处理 HTTP 上传请求
- 直接接收 `proto` 或 Kratos HTTP context
- 拼装前端响应
- 把 GORM Model 直接返回给 `biz`

补充要求：

- `data` 返回给 `biz` 的必须是 Domain Object
- GORM Model 只能存在于 `data` 层内部
- `data` 层日志必须使用统一注入的 `log.Helper`，禁止使用 `fmt.Println` 或 `log.Printf`
- `data/model` 的数据库主键类型是否与接口层完全一致，不要擅自改动；如无明确要求，优先保持模板现有模型风格
- 当 `service/biz` 使用 `int32`，而 `model` 仍为 `uint64` 或其他类型时，类型转换必须收敛在 `data` 层

### 错误与响应规范

必须统一遵守：

- 成功响应返回业务 `reply`
- 失败响应走统一错误结构
- 错误码集中定义在 `internal/errors/codes.go`
- 默认错误文案使用中文

成功响应必须满足：

- `code = 0`
- `message = 方法名 + " success"`
- `data = biz 返回的 Reply.Data`

失败响应必须满足：

- 统一走 `internal/errors`
- 不允许自行拼装另一套失败 JSON
- 不允许成功失败混用不同字段名

错误码单一来源规则：

- 当前模板默认以 `internal/errors/codes.go` 作为错误码定义源
- 如果项目后续改为在 `proto` 中定义错误码枚举，必须全项目统一迁移
- 禁止同时维护 `proto enum` 和 `internal/errors/codes.go` 两套业务错误码来源

文件上传接口补充要求：

- 文件格式错误、缺文件、大小超限等必须走统一错误规范
- 禁止为上传接口单独发明一套错误 JSON 结构

## AI 禁止生成清单

### 后端禁止项

- 普通接口跳过 `proto` 直接手写 HTTP 路由
- 普通接口使用 `/v1/users`、`/v1/users/{id}` 这类 REST 风格路径
- 在 `http.go` 中写业务逻辑
- `service` 直接操作数据库或对象存储
- `service` 直接依赖 `data`
- `biz` 接收 `http.Context`
- `biz` 写 SQL / GORM
- `data` 写业务判断
- `data` 拼装前端响应
- 复用通用 `UserReply`、`CommonReply`、`BaseReply`
- `service` 和 `biz` 方法名不一致
- `service` 返回中文 `"成功"` 或其他随意成功文案，而不是 `方法名 + success`
- `biz` 直接接收 `proto request`
- `biz` 返回完整 `reply`
