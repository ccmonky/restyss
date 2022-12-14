package restyss

import (
	"bufio"
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/cache"
	"github.com/pkg/errors"

	"github.com/t3/pkg/mock"
)

// NewMockerTransport 新建MockerTransport
func NewMockerTransport(matcher mock.Matcher, cache *cache.Codec, opts ...MockerTransportOption) *MockerTransport {
	mt := &MockerTransport{
		Matcher: matcher,
		cache:   cache,
	}
	for _, opt := range opts {
		opt(mt)
	}
	return mt
}

// MockerTransportOption MockerTransport属性设定函数
type MockerTransportOption func(*MockerTransport)

// WithExpiration 设定mock响应的过期时间
func WithExpiration(v time.Duration) MockerTransportOption {
	return func(mt *MockerTransport) {
		mt.Expiration = v
	}
}

// WithTransport 设定正常逻辑的Transport
func WithTransport(v *http.Transport) MockerTransportOption {
	return func(mt *MockerTransport) {
		mt.Transport = v
	}
}

// MockerTransport 带mock逻辑的Transport，即如果匹配Mock配置则返回mock响应，不匹配返回真实响应
type MockerTransport struct {
	Matcher   mock.Matcher
	Transport http.RoundTripper

	// Expiration is the cache expiration time.
	// Default expiration is 1 hour.
	Expiration time.Duration

	cache *cache.Codec
}

func (mt *MockerTransport) Provision() error {
	if mt.Transport == nil {
		mt.Transport = http.DefaultTransport
	}
	if mt.Expiration == 0 {
		mt.Expiration = time.Hour
	}
	if mt.Matcher == nil {
		return errors.Errorf("MockerTransport: nil mock matcher is not allowed")
	}
	if mt.cache == nil {
		return errors.Errorf("MockerTransport: nil cache is not allowed")
	}
	return nil
}

func (mt *MockerTransport) SetRoundTripper(rt http.RoundTripper) {
	mt.Transport = rt
}

// RoundTrip implements http.RoundTripper.
// 1. 识别动态开关
// 2. 拦截请求，计算eigenkey
// 3. 从cache获取响应，如果没有
// 4. 获取mocker，生成mock响应，缓存mock响应
func (mt *MockerTransport) RoundTrip(rq *http.Request) (*http.Response, error) {
	if _, exists := rq.Header[MockSwitchHeader.Name]; !exists { // NOTE：动态开关未打开
		return mt.Transport.RoundTrip(rq)
	}
	ek, mocker, err := mt.Matcher.Match(rq)
	if err != nil {
		return nil, errors.WithMessagef(err, "match mocker failed for %s", rq.URL.String())
	}
	if mocker == nil {
		return mt.Transport.RoundTrip(rq) // NOTE: 未匹配到ResponseMocker，即不mock此请求
	}
	var mockFunc func(*http.Request) (*http.Response, error)
	if mocker.IsTransparent() {
		mockFunc = mt.Transport.RoundTrip
	} else {
		mockFunc = mocker.Mock
	}
	fn := func() (interface{}, error) {
		rp, err := mockFunc(rq)
		if err != nil {
			return nil, errors.WithMessagef(err, "execute mock func failed for %s", rq.URL.String())
		}
		rpBuf := bufPool.Get().(*bytes.Buffer)
		rpBuf.Reset()
		defer bufPool.Put(rpBuf)
		err = rp.Write(rpBuf)
		if err != nil {
			return nil, errors.WithMessagef(err, "dump response failed for %s", rq.URL.String())
		}
		return rpBuf.Bytes(), nil
	}
	var dest []byte
	err = mt.cache.Once(&cache.Item{Key: ek, Object: &dest, Expiration: mt.Expiration, Func: fn})
	if err != nil {
		return nil, errors.WithMessagef(err, "get mock response from cache codec once failed for %s", rq.URL.String())
	}
	if dest == nil {
		return nil, errors.Errorf("get nil mock response for %s", rq.URL.String())
	}
	rp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(dest)), nil)
	rp.Header.Set(MockResponseHeader.Name, "true") // NOTE: 用于确认是mock响应！
	options := mocker.Extension()
	if options != nil && options.Latency.Duration > 0 {
		time.Sleep(options.Latency.Duration)
	}
	return rp, nil
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

var (
	_ http.RoundTripper = (*MockerTransport)(nil)
	_ TransportWrapper  = (*MockerTransport)(nil)
)
