package restyss

import (
	"context"
	"net/http"

	"github.com/ccmonky/typemap"
	"github.com/go-resty/resty/v2"
)

func R(a ...any) *resty.Request {
	var opts []RequestOption
	for _, p := range a {
		switch p := p.(type) {
		case string:
			opts = append(opts, WithClientName(p))
		case *http.Request:
			opts = append(opts, WithServerRequest(p))
		case RequestOption:
			opts = append(opts, p)
		}
	}
	options := &RequestOptions{}
	for _, opt := range opts {
		opt(options)
	}
	fn, err := typemap.Get[NewRequestFunc](context.Background(), options.NewRequestFuncName)
	if err != nil {
		panic(err)
	}
	request, err := fn(options)
	if err != nil {
		panic(err)
	}
	return request
}

type RequestOptions struct {
	ClientName         string
	ServerRequest      *http.Request
	Extension          map[string]any
	NewRequestFuncName string
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

func WithNewRequestFuncName(name string) RequestOption {
	return func(ros *RequestOptions) {
		ros.NewRequestFuncName = name
	}
}

type NewRequestFunc func(*RequestOptions) (*resty.Request, error)

func DefaultNewRequestFunc(options *RequestOptions) (*resty.Request, error) {
	client, err := typemap.Get[*resty.Client](context.Background(), options.ClientName)
	if err != nil {
		return nil, err
	}
	request := client.R()
	if options.ServerRequest != nil {
		request.SetContext(options.ServerRequest.Context())
	}
	return request, nil
}
