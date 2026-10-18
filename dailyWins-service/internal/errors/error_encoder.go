package errors

import (
	nethttp "net/http"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)


// 将错误 写到http响应中
func EncodeHTTPError(w nethttp.ResponseWriter, r *nethttp.Request, err error) {
	// 将任意错误转换成统一的HTTPError
	se := FromError(err)
	// 序列化成json
	codec, _ := kratoshttp.CodecForRequest(r, "Accept")
	body, marshalErr := codec.Marshal(se)
	if marshalErr != nil {
		w.WriteHeader(nethttp.StatusInternalServerError)
		return
	}
	// 将body返回给客户端
	w.Header().Set("Content-Type", "application/"+codec.Name())
	if se.HttpCode >= 100 && se.HttpCode < 600 {
		w.WriteHeader(se.HttpCode)
	} else {
		w.WriteHeader(nethttp.StatusInternalServerError)
	}
	_, _ = w.Write(body)
}
