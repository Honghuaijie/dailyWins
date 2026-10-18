# API Proto

将对外 API 的 proto 文件放在 `api/<domain>/vN/*.proto`。

示例：
- `api/user/v1/user.proto`
- `api/post/v1/post.proto`

生成结果输出到 `api/all-pb-go/vN/`，项目内部统一引用对应版本目录。

命令：
- `make api`
