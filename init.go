package restyss

import (
	"context"

	"github.com/ccmonky/typemap"
	"github.com/go-resty/resty/v2"
)

func init() {
	typemap.MustRegisterType[*resty.Client]()
	typemap.MustRegister[*resty.Client](context.Background(), "", New())

	typemap.MustRegisterType[NewRequestFunc]()
	typemap.MustRegister[NewRequestFunc](context.Background(), "", DefaultNewRequestFunc)
}
