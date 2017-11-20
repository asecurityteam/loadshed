package transport

import (
	"fmt"
	"net/http"
)

// errCodeTransport is an HTTP client wrapper that returns error if the Response
// contains one of the err codes
type errCodeTransport struct {
	wrapped    http.RoundTripper
	errorCodes []int
}

// RoundTrip circuit breaks the outgoing request if needed and calls the wrapped Client.
func (c *errCodeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var er error

	resp, er = c.wrapped.RoundTrip(r)

	if er != nil {
		return resp, er
	}
	for _, code := range c.errorCodes {
		if code == resp.StatusCode {
			return resp, &codeError{errCode: code} // return error if matches expected error codes
		}
	}
	return resp, er

}

// NewErrCodes returns a transport wrapper around the error codes transport
func NewErrCodes(errorCodes []int) func(http.RoundTripper) http.RoundTripper {
	return func(c http.RoundTripper) http.RoundTripper {
		return &errCodeTransport{
			wrapped:    c,
			errorCodes: errorCodes,
		}
	}
}

type codeError struct {
	errCode int
}

func (c *codeError) Error() string {
	return fmt.Sprintf("error code is %d", c.errCode)
}
