package transport

import (
	"context"
	"net/http"

	"bitbucket.org/atlassian/loadshed"
	"bitbucket.org/atlassian/rolling"
)

// Option is a wrapper around the transport modifying its behavior
type Option func(*Transport) *Transport

// ErrorCodes option wraps the errorcodes transport
func ErrorCodes(errorCodes []int) Option {
	return func(t *Transport) *Transport {
		var et = NewErrCodes(errorCodes)
		t.wrapped = et(t.wrapped)
		return t
	}
}

// Callback option adds a callback for error
func Callback(cb func(*http.Request) (*http.Response, error)) Option {
	return func(t *Transport) *Transport {
		t.callback = cb
		return t
	}
}

// Transport is an HTTP client wrapper that provides circuit breaker functionality for
// the outgoing request.
type Transport struct {
	wrapped  http.RoundTripper
	callback func(*http.Request) (*http.Response, error)
	load     loadshed.Doer
}

// RoundTrip circuit breaks the outgoing request if needed and calls the wrapped Client.
func (c *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var e = c.load.Do(func() error {
		var innerResp, innerEr = c.wrapped.RoundTrip(r)
		if innerEr != nil {
			return innerEr
		}
		resp = innerResp
		return nil
	})

	switch e.(type) {
	case loadshed.Rejected:
		r = r.WithContext(NewContext(r.Context(), e.(loadshed.Rejected).Aggregate))
		if c.callback != nil {
			return c.callback(r)
		}
	}

	return resp, e
}

// New takes in a loadshed Doer and transport options and returns a RoundTripper wrapper
func New(l loadshed.Doer, options ...Option) func(c http.RoundTripper) http.RoundTripper {
	return func(c http.RoundTripper) http.RoundTripper {
		var t = &Transport{wrapped: c, load: l}
		for _, option := range options {
			t = option(t)
		}
		return t
	}
}

type ctxKey string

var key = ctxKey("loadshedtransport")

// NewContext inserts an aggregate into the context after a request has been
// rejected.
func NewContext(ctx context.Context, val *rolling.Aggregate) context.Context {
	return context.WithValue(ctx, key, val)
}

// FromContext extracts an aggregate from the context after a request has
// been rejected.
func FromContext(ctx context.Context) *rolling.Aggregate {
	if v, ok := ctx.Value(key).(*rolling.Aggregate); ok {
		return v
	}
	return nil
}
