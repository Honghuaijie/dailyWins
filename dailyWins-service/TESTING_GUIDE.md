# Testing Guide

这份文档说明当前模板推荐的测试分层、命名方式和最小测试样例结构，方便后续基于模板创建的新项目直接照着补测试。

## 测试文件命名

- Go 测试文件必须以 `_test.go` 结尾
- 测试函数必须以 `Test` 开头，例如：

```go
func TestUserUsecaseCreate_MissingNameOrEmail(t *testing.T) {}
```

## 常用命令

运行全部模板测试：

```bash
make api
make test
```

只运行 `biz` 层测试：

```bash
make test-biz
```

只运行某一条测试：

```bash
go test ./internal/biz -run TestUserUsecaseCreate_MissingNameOrEmail -v
```

## 分层建议

### `biz` 层

`biz` 层主要测试业务规则：

- 资源不存在
- 正常成功
- 返回的 `Reply.Data` 是否符合预期
- 错误码是否符合预期

推荐方式：

- 不连接真实数据库
- 使用 fake repo / fake tx
- 通过 `internal/errors.FromError(err)` 断言 `Code / HttpCode / Message`

参考样例：

- `internal/biz/user_test.go`

### `service` 层

`service` 层主要测试协议转换：

- 请求参数校验
- 是否原样透传 `biz` 错误
- 是否统一封装成功响应的 `code / message / data`
- 列表接口是否正确转换 repeated 字段和 `total`

参考样例：

- `internal/service/user_test.go`

### `server/http` 层

HTTP 层主要测试最终接口表现：

- HTTP 状态码是否正确
- 成功响应是否符合预期
- 统一错误响应是否生效
- 健康检查接口是否可用

参考样例：

- `internal/server/http_test.go`

## fake 依赖原则

模板阶段统一采用手写 fake，而不是引入 mock 框架：

- `fakeUserRepo`
- `fakeTx`

这样做的好处：

- 依赖少
- 容易看懂
- 适合作为教学样例

## 错误断言规范

测试里不要只判断 `err != nil`，而要尽量断言统一错误结构：

```go
se := errors.FromError(err)
if se.Code != errors.ErrInvalidID.Code {
	t.Fatalf("unexpected code: got %d want %d", se.Code, errors.ErrInvalidID.Code)
}
```

推荐至少断言：

- `Code`
- `HttpCode`
- 关键场景下的 `Message`

## 推荐补充顺序

如果你是第一次在这个模板里补测试，建议按这个顺序：

1. `service` 层参数校验失败
2. `biz` 层正常成功
3. `biz` 层资源不存在
4. `service` 层错误透传
5. `service` 层成功响应封装
6. HTTP 失败样例
7. HTTP 成功样例

## 当前模板内置测试样例

### `internal/biz/user_test.go`

- `TestUserUsecaseCreate_Success`
- `TestUserUsecaseGet_UserNotFound`

### `internal/service/user_test.go`

- `TestUserServiceCreateUser_ReturnsBizError`
- `TestUserServiceCreateUser_Success`
- `TestUserServiceListUsers_Success`

### `internal/server/http_test.go`

- `TestHTTPCreateUser_MissingNameOrEmail`
- `TestHTTPCreateUser_Success`
- `TestHTTPHealth_Success`
