package restyss

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ccmonky/typemap"
	"github.com/go-resty/resty/v2"
)

func R(ps ...any) *resty.Request {
	var opts []RequestOption
	for _, p := range ps {
		switch p := p.(type) {
		case string:
			opts = append(opts, WithClientName(p))
		case *http.Request:
			opts = append(opts, WithServerRequest(p))
		case RequestOption:
			opts = append(opts, p)
		default:
			panic(fmt.Errorf("restyss.R not support parameter type: %T", p))
		}
	}
	options := &RequestOptions{
		RequestHookNames: []string{DefaultRequestHookName},
	}
	for _, opt := range opts {
		opt(options)
	}
	client, err := typemap.Get[*resty.Client](context.Background(), options.ClientName)
	if err != nil {
		panic(err)
	}
	request := client.R()
	for _, hookName := range options.RequestHookNames { // FIXME: typemap.GetMany!!!
		fn, err := typemap.Get[RequestHook](context.Background(), hookName)
		if err != nil {
			panic(err)
		}
		err = fn(request, options)
		if err != nil {
			panic(err)
		}
	}
	return request
}

type RequestOptions struct {
	ClientName       string
	RequestHookNames []string
	ServerRequest    *http.Request
	Extension        map[string]any
}

type RequestOption func(*RequestOptions)

func WithClientName(name string) RequestOption {
	return func(ros *RequestOptions) {
		ros.ClientName = name
	}
}

func WithServerRequest(r *http.Request) RequestOption {
	return func(ros *RequestOptions) {
		ros.ServerRequest = r
	}
}

func WithExtestion(k string, v any) RequestOption {
	return func(ros *RequestOptions) {
		ros.Extension[k] = v
	}
}

func WithRequestHookNames(names []string) RequestOption {
	return func(ros *RequestOptions) {
		ros.RequestHookNames = names
	}
}

func AppendRequestHook(name string) RequestOption {
	return func(ros *RequestOptions) {
		ros.RequestHookNames = append(ros.RequestHookNames, name)
	}
}

// RequestHook used to custom request with external input
type RequestHook func(*resty.Request, *RequestOptions) error

func DefaultRequestHook(request *resty.Request, options *RequestOptions) error {
	if options.ServerRequest != nil {
		request.SetContext(options.ServerRequest.Context())
	}
	return nil
}
