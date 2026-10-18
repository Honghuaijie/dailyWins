package errors

import (
	stderrors "errors"
	"fmt"
	nethttp "net/http"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)
// 这个文件用于生成错误对象error，用于给业务端返回
type HTTPError struct {
	HttpCode int         `json:"-"`
	Code     int         `json:"code"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data,omitempty"`
}

// 使用默认message
func E(def CodeDef) *HTTPError {
	return newHTTPError(def, def.Message, nil)
}

// 业务自定义message
func EWithMessage(def CodeDef, msg string) *HTTPError {
	return newHTTPError(def, msg, nil)
}

// 使用默认message，并携带data
func EWithData(def CodeDef, data interface{}) *HTTPError {
	return newHTTPError(def, def.Message, data)
}

// 业务自定义message 和data
func EWithMessageData(def CodeDef, msg string, data interface{}) *HTTPError {
	return newHTTPError(def, msg, data)
}


// 输出错误结构体的 code 和message
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTPError:%d %s", e.HttpCode, e.Message)
}

// 错误兜底转换。作用是：不管传进来的是什么 error，尽量统一转换成自己的 HTTPError
func FromError(err error) *HTTPError {
	if err == nil {
		return nil
	}

	var httpErr *HTTPError
	if stderrors.As(err, &httpErr) {
		return httpErr
	}

	var kratosErr *kerrors.Error
	if stderrors.As(err, &kratosErr) {
		def := mapKratosError(kratosErr)
		if kratosErr.Message != "" {
			return EWithMessage(def, kratosErr.Message)
		}
		return E(def)
	}

	// 如果不能分析出这个错误，那么就返回服务器内部错误
	return E(ErrInternal)
}

// 新建HTTPError
func newHTTPError(def CodeDef, message string, data interface{}) *HTTPError {
	if message == "" {
		message = def.Message
	}
	return &HTTPError{
		HttpCode: def.HTTPCode,
		Code:     def.Code,
		Message:  message,
		Data:     data,
	}
}

// 列举常见的http错误，并映射成我们自定义的错误
func mapKratosError(err *kerrors.Error) CodeDef {
	switch err.Code {
	case nethttp.StatusBadRequest:
		return ErrInvalidParam
	case nethttp.StatusUnauthorized:
		return ErrUnauthorized
	case nethttp.StatusForbidden:
		return ErrForbidden
	case nethttp.StatusNotFound:
		return ErrNotFound
	case nethttp.StatusConflict:
		return ErrConflict
	case nethttp.StatusTooManyRequests:
		return ErrTooManyRequests
	case nethttp.StatusBadGateway:
		return ErrUpstreamRequestFailed
	case nethttp.StatusServiceUnavailable:
		return ErrDependencyUnavailable
	case nethttp.StatusGatewayTimeout:
		return ErrRequestTimeout
	default:
		return ErrInternal
	}
}
