package restyss

import "github.com/go-resty/resty/v2"

// CtxResponseWriterKey 用于服务端和客户端之间传递ResponseWriter
type CtxResponseWriterKey struct{}

// CtxProxySetHeadersKey 用于服务端和上游之间透传的请求头
type CtxProxySetHeadersKey struct{}

// CtxHeaderKey 用于服务端和客户端之间传递特定请求头
// - 上下文注入：通过`http.handlers.context`完成服务请求头注入
// - 上下文传递：http.Handler中通过httpx.GetRestyV2("xxx").R().SetContext(r.Contxt())....传递上下文
// - 上下文提取：httpx.Client在相关的中间件内提取上下文是否包含特定信息执行特定逻辑
// Note: 通常用于此头信息不应透传到上游服务，仅通过Context传递和使用，如果可以或应该传递给上游，使用CtxProxySetHeadersKey即可
type CtxHeaderKey string

// ProxySetHeadersRequestMiddleware 自动从context提取CtxProxySetHeadersKey指定的headers并透明传递
func ProxySetHeadersRequestMiddleware(c *resty.Client, rq *resty.Request) error {
	if value := rq.Context().Value(CtxProxySetHeadersKey{}); value != nil {
		headers, ok := value.(map[string]string)
		if ok {
			rq.SetHeaders(headers)
		}
	}
	return nil
}

/*
var (
	// CtxMockSwitchHeaderKey 当前通过透传头实现Mock，此Key未使用，因为MockTransport未使用Context
	CtxMockSwitchHeaderKey CtxHeaderKey = CtxHeaderKey(MockSwitchHeader.Name)

	// CtxMockTagHeaderKey 当前通过透传头实现Mock，此Key未使用，因为mock matcher未使用Context
	CtxMockTagHeaderKey CtxHeaderKey = CtxHeaderKey(MockTagHeader.Name)
)
*/
