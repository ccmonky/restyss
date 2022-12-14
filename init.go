package restyss

import (
	"context"

	"github.com/ccmonky/typemap"
	"github.com/go-resty/resty/v2"
)

const (
	DefaultClientName      = ""
	DefaultRequestHookName = ""
)

func init() {
	typemap.MustRegisterType[*resty.Client]()
	typemap.MustRegister[*resty.Client](context.Background(), DefaultClientName, New())

	typemap.MustRegisterType[RequestHook]()
	typemap.MustRegister[RequestHook](context.Background(), DefaultRequestHookName, DefaultRequestHook)

	typemap.MustRegisterType[resty.RequestMiddleware]()
	typemap.MustRegisterType[resty.ResponseMiddleware]()
}
