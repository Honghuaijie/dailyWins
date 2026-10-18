package errors

import nethttp "net/http"

// CodeDef 定义对外暴露的稳定错误码、默认文案和 HTTP 状态码。
//
// 说明：
// 1. Code 是业务错误码，给前端、日志、监控和排障使用。
// 2. HTTPCode 是 HTTP 协议状态码，给浏览器、网关、HTTP 客户端使用。
// 3. Message 是默认中文文案；如果某个场景需要更具体的提示，可以用 EWithMessage 覆盖。
//
// 使用原则：
// - 业务层返回错误时，优先从本文件选择最贴近语义的错误枚举，不要直接手写裸数字。
// - 如果只是“同一类错误在不同场景下提示文案不同”，保留原错误码，用 EWithMessage 覆盖文案即可。
// - 如果现有错误码无法准确表达语义，再在本文件中追加新的错误定义。
//
// 号段约定：
// - 0：成功
// - 40000-49999：请求侧、鉴权、权限、资源、业务冲突类错误
// - 50000-59999：服务内部、配置、依赖、超时类错误
type CodeDef struct {
	Code     int
	Message  string
	HTTPCode int
}

var (
	// 成功类。
	// 一般不会主动在业务代码里 return ErrOK，它主要用于说明“0 表示成功”这条约定。
	ErrOK = CodeDef{Code: 0, Message: "成功", HTTPCode: nethttp.StatusOK}

	// 参数校验类。
	// 用法：
	// - ErrInvalidParam：参数整体不合法，但不想细分到某个字段时使用。
	// - ErrMissingRequiredField：缺少必填字段，通常配合 EWithMessage 指出具体字段。
	// - ErrInvalidID：id <= 0、路径 id 非法、主键格式错误时使用。
	// - ErrInvalidRequestBody：请求体 JSON/XML 结构错误、字段类型错误时使用。
	// - ErrInvalidQueryParam：query 参数格式错误、枚举值非法、分页参数非法时使用。
	ErrInvalidParam         = CodeDef{Code: 40000, Message: "请求参数错误", HTTPCode: nethttp.StatusBadRequest}
	ErrMissingRequiredField = CodeDef{Code: 40001, Message: "缺少必填字段", HTTPCode: nethttp.StatusBadRequest}
	ErrInvalidID            = CodeDef{Code: 40002, Message: "无效的ID", HTTPCode: nethttp.StatusBadRequest}
	ErrInvalidRequestBody   = CodeDef{Code: 40003, Message: "请求体错误", HTTPCode: nethttp.StatusBadRequest}
	ErrInvalidQueryParam    = CodeDef{Code: 40004, Message: "查询参数错误", HTTPCode: nethttp.StatusBadRequest}

	// 认证和权限类。
	// 用法：
	// - ErrUnauthorized：只知道“未登录/认证失败”，但不区分具体原因时使用。
	// - ErrMissingToken：请求缺少 token、session、签名头时使用。
	// - ErrInvalidToken：token 非法、签名校验失败时使用。
	// - ErrExpiredToken：token 已过期时使用。
	// - ErrForbidden：用户已登录，但没有当前资源/操作权限时使用。
	ErrUnauthorized = CodeDef{Code: 40100, Message: "未登录或认证失败", HTTPCode: nethttp.StatusUnauthorized}
	ErrMissingToken = CodeDef{Code: 40101, Message: "缺少认证令牌", HTTPCode: nethttp.StatusUnauthorized}
	ErrInvalidToken = CodeDef{Code: 40102, Message: "认证令牌无效", HTTPCode: nethttp.StatusUnauthorized}
	ErrExpiredToken = CodeDef{Code: 40103, Message: "认证令牌已过期", HTTPCode: nethttp.StatusUnauthorized}
	ErrForbidden    = CodeDef{Code: 40300, Message: "无权限访问", HTTPCode: nethttp.StatusForbidden}

	// 资源和冲突类。
	// 用法：
	// - ErrNotFound：资源不存在，但不想暴露具体资源类型时使用。
	// - ErrUserNotFound：示例业务中的具体资源不存在；新项目可以继续定义 ErrOrderNotFound 等。
	// - ErrConflict：资源状态冲突，但未细分原因时使用。
	// - ErrDuplicate：唯一键冲突、重复创建、重复绑定时使用。
	// - ErrTooManyRequests：限流、频率超限、操作过于频繁时使用。
	ErrNotFound        = CodeDef{Code: 40400, Message: "资源不存在", HTTPCode: nethttp.StatusNotFound}
	ErrUserNotFound    = CodeDef{Code: 40401, Message: "用户不存在", HTTPCode: nethttp.StatusNotFound}
	ErrConflict        = CodeDef{Code: 40900, Message: "资源冲突", HTTPCode: nethttp.StatusConflict}
	ErrDuplicate       = CodeDef{Code: 40901, Message: "资源已存在", HTTPCode: nethttp.StatusConflict}
	ErrTooManyRequests = CodeDef{Code: 42900, Message: "请求过于频繁", HTTPCode: nethttp.StatusTooManyRequests}

	// 服务端和依赖类。
	// 用法：
	// - ErrInternal：兜底内部错误；不希望把底层实现细节暴露给客户端时使用。
	// - ErrUnknownBusiness：业务流程失败，但暂时还没有更合适的业务错误码时使用。
	// - ErrDBOperation：数据库读写、事务提交、更新删除失败时使用。
	// - ErrDBConnection：数据库连接建立失败、连接池不可用时使用。
	// - ErrConfigInvalid：服务启动配置缺失、配置值非法时使用。
	// - ErrSerialization：序列化/反序列化失败，或对象编码失败时使用。
	// - ErrUpstreamRequestFailed：调用外部 HTTP/gRPC 服务返回失败时使用。
	// - ErrDependencyUnavailable：Redis、MQ、第三方服务当前不可用时使用。
	// - ErrRequestTimeout：调用下游超时、任务执行超时时使用。
	ErrInternal              = CodeDef{Code: 50000, Message: "服务内部错误", HTTPCode: nethttp.StatusInternalServerError}
	ErrUnknownBusiness       = CodeDef{Code: 50001, Message: "未知业务错误", HTTPCode: nethttp.StatusInternalServerError}
	ErrDBOperation           = CodeDef{Code: 50010, Message: "数据库操作失败", HTTPCode: nethttp.StatusInternalServerError}
	ErrDBConnection          = CodeDef{Code: 50011, Message: "数据库连接失败", HTTPCode: nethttp.StatusInternalServerError}
	ErrConfigInvalid         = CodeDef{Code: 50020, Message: "服务配置错误", HTTPCode: nethttp.StatusInternalServerError}
	ErrSerialization         = CodeDef{Code: 50030, Message: "序列化失败", HTTPCode: nethttp.StatusInternalServerError}
	ErrUpstreamRequestFailed = CodeDef{Code: 50200, Message: "上游请求失败", HTTPCode: nethttp.StatusBadGateway}
	ErrDependencyUnavailable = CodeDef{Code: 50300, Message: "依赖服务不可用", HTTPCode: nethttp.StatusServiceUnavailable}
	ErrRequestTimeout        = CodeDef{Code: 50400, Message: "请求超时", HTTPCode: nethttp.StatusGatewayTimeout}
)
