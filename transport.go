package restyss

import "net/http"

type TransportWrapper interface {
	http.RoundTripper

	// SetRoundTripper set the underline http.RoundTripper of the wrapper, panic if error
	SetRoundTripper(http.RoundTripper)
}

// Chain is a helper function for composing TransportWrappers. Requests will
// traverse them in the order they're declared. That is, the first TransportWrapper
// is treated as the outermost TransportWrapper.
func TransportWrapperChain(wrappers ...TransportWrapper) http.RoundTripper {
	if len(wrappers) == 0 {
		return http.DefaultTransport
	}
	for i := len(wrappers) - 2; i >= 0; i-- { // reverse
		wrappers[i].SetRoundTripper(wrappers[i+1])
	}
	return wrappers[0]
}
