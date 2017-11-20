package middleware

import (
	"context"
	"fmt"
	"net/http"

	"bitbucket.org/atlassian/loadshed"
	"bitbucket.org/atlassian/rolling"
)

// Option is a wrapper for middleware
type Option func(*Middleware) *Middleware

// Callback Option adds a callback to the middleware
func Callback(cb http.Handler) Option {
	return func(m *Middleware) *Middleware {
		m.callback = cb
		return m
	}
}

// ErrCodes Option adds errcode check to the middleware
func ErrCodes(errCodes []int) Option {
	return func(m *Middleware) *Middleware {
		m.errCodes = errCodes
		return m
	}
}

// Middleware struct represents a loadshed middleware
type Middleware struct {
	next     http.Handler
	errCodes []int
	load     loadshed.Doer
	callback http.Handler
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var proxy = wrapWriter(w)

	var lerr = m.load.Do(func() error {
		m.next.ServeHTTP(proxy, r)
		for _, errCode := range m.errCodes {
			if proxy.Status() == errCode {
				return &codeError{errCode: proxy.Status()}
			}
		}
		return nil
	})

	switch lerr.(type) {
	case loadshed.Rejected:
		r = r.WithContext(NewContext(r.Context(), lerr.(loadshed.Rejected).Aggregate))
		m.callback.ServeHTTP(proxy, r)
	}
}

func defaultCallback(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

// New takes in options and returns a wrapped middleware
func New(l loadshed.Doer, options ...Option) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		var m = &Middleware{
			next:     next,
			load:     l,
			callback: http.HandlerFunc(defaultCallback),
		}
		for _, option := range options {
			m = option(m)
		}
		return m
	}
}

type ctxKey string

var key = ctxKey("loadshedmiddleware")

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

type codeError struct {
	errCode int
}

func (c *codeError) Error() string {
	return fmt.Sprintf("error code is %d", c.errCode)
}
