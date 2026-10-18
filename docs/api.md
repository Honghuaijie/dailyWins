# Daily Wins V1 接口

唯一接口契约源：`dailyWins-service/api/dailywins/v1/dailywins.proto`。HTTP 由 Kratos 生成注册；所有普通接口只使用 GET/POST。响应 JSON 使用 lowerCamelCase，成功均为 `{"code":0,"message":"方法名 success","data":{...}}`；错误使用相同的 code/message 结构和正确的 HTTP 状态。

## 认证与请求保护

- 所有 POST：`Content-Type: application/json`、`X-Daily-Wins: 1`，参数在 JSON body，不允许 POST query 参数。body 最大 64 KiB，拒绝未知字段及非法 JSON 类型。
- 浏览器登录：`POST /v1/public/auth/login`，`{"username":"你的账号","password":"你的密码"}`。返回 account、expiresAt；token 为空字符串。凭证写入 `daily_wins_session` Cookie：HttpOnly、SameSite=Lax、Path=/，生产 Secure。
- App 登录：同一路由，显式增加 `"bearer":true`，返回非空 token；后续使用 `Authorization: Bearer <token>`。App 应使用系统安全凭证存储。
- JWT HS256 + 随机会话标识，数据库保存会话标识的 SHA-256。每次请求同时校验 JWT 与数据库会话。默认绝对有效期 168 小时，不自动续期；AUTH_TTL_HOURS 可配 1–720。
- `GET /v1/user/auth/me`：返回 `data.account={id,username}`；缺少、过期或已撤销凭证返回 401。
- `POST /v1/user/auth/logout`，body `{}`：撤销当前会话并清 Cookie。其他设备会话不受影响；已撤销凭证再次访问返回 401。并发中已通过认证的请求可能已在执行，前端会先等待保存完成再退出。
- 不开放注册；模板 `/v1/users/*` 未注册，gRPC 不注册业务接口。
- 登录按实际 TCP 来源地址限制：15 分钟窗口最多 5 次尝试，成功清零；第 6 次返回 429。接口总限流每来源每分钟 300 次。V1 为单实例内存限流，重启重置；反向代理后的访客共用代理来源，不信任任意 X-Forwarded-For。多实例或公网入口应在可信网关按真实客户端追加限流。
- 默认同源。跨域来源只接受 ALLOWED_ORIGINS 中的完整来源，不接受通配符；未知来源返回 403。HTTPS 在代理终止时，将实际站点来源加入该列表。

## 任务 DTO

```json
{
  "id": "7d35ea08-69e0-4bc6-8aa6-f36ec9bc0375",
  "userId": 1,
  "date": "2026-10-17",
  "name": "阅读一章",
  "note": "留一点感受",
  "done": true,
  "createdAt": "2026-10-17T02:00:00Z",
  "updatedAt": "2026-10-17T02:00:00Z",
  "version": 1,
  "deleted": false
}
```

`date` 为纯日期 YYYY-MM-DD，支持 0001-01-01 至 9999-12-31；不可传时间戳，不做时区转换。时间戳单独使用 RFC3339 UTC。名称去除首尾空白后必填，原输入最多 200 个 Unicode 字符；备注最多 4000 字。任务总数不限。

用户归属始终来自认证会话，不能在请求中指定 userId。查询、更新、删除、恢复均限定所属用户。

## 路由

| 方法及路径 | 请求字段 | data |
|---|---|---|
| GET `/v1/user/tasks/list?from=2026-10-01&to=2026-11-08` | from/to，包含两端，跨度最大 366 天 | items、days |
| POST `/v1/user/tasks/create` | requestId、id、date、name、note（可空）、done | task |
| POST `/v1/user/tasks/update` | requestId、id、version、name、note、done | task |
| POST `/v1/user/tasks/move` | requestId、id、version、date | task |
| POST `/v1/user/tasks/delete` | requestId、id、version | task（deleted=true） |
| POST `/v1/user/tasks/restore` | requestId、id、version | task（deleted=false） |

新建任务的 id 和每次操作的 requestId 都由客户端生成标准小写 UUID。`update` 为完整编辑快照，name/note/done 应全部传入，省略 note/done 按 proto 默认空字符串/false 处理。移动只改变日期且只允许当前未完成任务。恢复保留 ID、原日期、名称、备注和完成状态。

查询结果按日期、创建时间、ID 排序。`days` 仅包含有有效任务的日期，每项 `{date,completed,achieved}`，achieved 始终为当前 completed >= 3；无记录的日期按完成 0 项处理。删除与取消完成都会重新计算，未完成任务不自动顺延。

## 幂等、并发与撤销

每次成功写入 version 加一。修改、移动、删除、恢复必须携带当前 version；旧版本返回 HTTP 409，不会覆盖新记录。

请求失败/响应丢失后，用**相同 requestId 和完全相同 body**重试。服务器在同一事务内保存任务和幂等快照，重复请求返回原结果，不重复写入；相同 requestId 改传不同内容返回 409。幂等记录目前保留，不自动清理。按用户行锁串行化写入，适合固定个人账号规模。

删除采用逻辑删除，列表不返回已删除项；删除响应带新 version，撤销用此 version 调用 restore。前端提供 5 秒撤销入口，服务端没有 5 秒恢复期限。移动不是复制。

## 错误

| HTTP | code | 含义 |
|---|---|---|
| 400 | 40000/40003 | 字段、日期、范围、UUID、长度、JSON 或版本参数不合法 |
| 401 | 40100–40103 | 登录失败、凭证缺失、无效或过期/撤销 |
| 403 | 40300 | 来源不允许或缺少写请求保护头 |
| 404 | 40400 | 任务不存在、不属于当前账号，或已删除而不能执行该动作 |
| 409 | 40900/40901 | 版本冲突、重复 ID、请求 ID 内容冲突、已完成任务不能移动 |
| 429 | 42900 | 登录或请求频率限制 |
| 500 | 50000/50010 | 内部或数据库错误；不返回数据库连接或 SQL 细节 |

前端失败时保留任务与草稿，不显示已保存；401 重新登录后继续原请求，409 不自动覆盖其他编辑，提供导出未保存内容并重新加载入口。`GET /health` 仅说明进程存活，不证明数据库可用。
