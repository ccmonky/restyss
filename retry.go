package restyss

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/go-resty/resty/v2"
)

// MakeRestyRetryPolicy httpx的resty v2的重试策略函数生成器
func MakeRestyRetryPolicy(options RetryOptions) resty.RetryConditionFunc {
	return func(rp *resty.Response, err error) bool {
		// do not retry on context.Canceled or context.DeadlineExceeded
		if rp != nil && rp.Request != nil && rp.Request.Context() != nil {
			if rp.Request.Context().Err() != nil {
				return false
			}
		}
		var resp *http.Response
		if rp != nil && rp.RawResponse != nil {
			resp = rp.RawResponse
		}
		shouldRetry, _ := baseRetryPolicy(resp, err, options)
		return shouldRetry
	}
}

// RetryOptions 重试设定项
type RetryOptions struct {
	IdempotentMethods   []string
	NoRetryErrors       []error
	NoRetryErrorRegexps []string

	noRetryErrorRegexps []*regexp.Regexp
}

// Provision 初始化
func (ro *RetryOptions) Provision() error {
	for _, regstr := range ro.NoRetryErrorRegexps {
		reg, err := regexp.Compile(regstr)
		if err != nil {
			return err
		}
		ro.noRetryErrorRegexps = append(ro.noRetryErrorRegexps, reg)
	}
	return nil
}

// original work from `https://github.com/hashicorp/go-retryablehttp` with some modifications
func baseRetryPolicy(resp *http.Response, err error, options RetryOptions) (bool, error) {
	// 如果设定了幂等方法集合，则判断请求是否幂等，非幂等不重试
	// FIXME: 非幂等请求返回408, 429, 500, 502和504是否应该重试？
	if !isIdempotent(resp.Request, options.IdempotentMethods...) {
		return false, nil
	}

	if err != nil {
		// Don't retry if the error was due to too many redirects.
		// Modified by yunfei.liu: 从下面的断言中移出，因为对resty也试用，但是resty不是*url.Error!
		if redirectsErrorRe.MatchString(err.Error()) {
			return false, err
		}

		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, v
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, v
			}
		}

		// 其他自定义非重试错误判断
		for _, noRetryError := range options.NoRetryErrors { // added by yunfei.liu
			if errors.Is(err, noRetryError) {
				return false, err
			}
		}

		// 自定义不可重试的错误信息正则匹配
		for _, reg := range options.noRetryErrorRegexps {
			if reg != nil && reg.MatchString(err.Error()) {
				return false, err
			}
		}

		// The error is likely recoverable so retry.
		return true, nil
	}

	if resp == nil { // added by yunfei.liu
		return false, nil
	}

	// 429 Too Many Requests is recoverable. Sometimes the server puts
	// a Retry-After response header to indicate when the server is
	// available to start processing request from client.
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	// The HyperText Transfer Protocol (HTTP) 408 Request Timeout response status code means that
	// the server would like to shut down this unused connection.
	// It is sent on an idle connection by some servers, even without any previous request by the client.
	if resp.StatusCode == http.StatusRequestTimeout { // added by yunfei.liu
		return true, nil
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	//
	// 501 Not Implemented. The server does not support the functionality
	// required to fulfill the request. This is the appropriate response
	// when the server does not recognize the request method
	// and is not capable of supporting it for any resource.
	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	return false, nil
}

func isIdempotent(r *http.Request, idempotentMethods ...string) bool {
	if r == nil {
		return false
	}
	if len(idempotentMethods) == 0 {
		return true
	}
	// The Idempotency-Key, while non-standard, is widely used to
	// mean a POST or other request is idempotent. See
	// https://golang.org/issue/19943#issuecomment-421092421
	if _, ok := r.Header["Idempotency-Key"]; ok {
		return true
	}
	if _, ok := r.Header["X-Idempotency-Key"]; ok {
		return true
	}
	if Contains(idempotentMethods, r.Method) {
		return true
	}
	return false
}

// Contains check if element in list, used to test
func Contains(list []string, elem string) bool {
	for _, t := range list {
		if t == elem {
			return true
		}
	}
	return false
}

var (
	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)
)
