package transport

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/asecurityteam/loadshed"
)

type fixtureTransport struct {
	Response *http.Response
	Err      error
}

func (c *fixtureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return c.Response, c.Err
}

func TestTransport(t *testing.T) {

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	var load = &fakeLoadShedder{}
	var tr = New(load)(wrapped)

	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
	var _, err = tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("Unexpected error %s", err.Error())
	}
}

func TestTransportErrorRejected(t *testing.T) {

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}
	var cerr = errors.New("test")
	var load = &fakeLoadShedder{err: cerr}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	var tr = New(load)(wrapped)

	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
	var _, err = tr.RoundTrip(req)
	if err == nil {
		t.Fatal("Did not get expected error ")
	}

}

func TestTransportErrCodeOption(t *testing.T) {
	var load = &fakeLoadShedder{}
	resp := &http.Response{
		Status:     "OK",
		StatusCode: 500,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	var errOption = ErrorCodes([]int{500, 501})
	var tr = New(load, errOption)(wrapped)

	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
	var err error
	_, err = tr.RoundTrip(req)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
}

func TestTransportOptionCallback(t *testing.T) {

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}
	var cerr = loadshed.Rejected{}
	var load = &fakeLoadShedder{err: cerr}

	var counter = 0
	var cb = func(*http.Request) (*http.Response, error) {
		counter++
		return nil, cerr
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	var cbOption = Callback(cb)
	var tr = New(load, cbOption)(wrapped)

	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
	var _, err = tr.RoundTrip(req)
	if err == nil {
		t.Fatal("Did not get expected error ")
	}
	if counter != 1 {
		t.Fatal("Callback not called")
	}

}

func TestTransportLoadshedder(t *testing.T) {

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 500,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	var lower = 50.0
	var upper = 50.0
	var bucketSize = time.Millisecond
	var buckets = 5
	var preallocHint = 5
	var requiredPoints = 1
	var errorCodes = []int{500}
	var load = loadshed.New(loadshed.ErrorRate(lower, upper, bucketSize, buckets, preallocHint, requiredPoints))
	var tr = New(load, ErrorCodes(errorCodes))(wrapped)

	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
	var err error
	_, err = tr.RoundTrip(req)
	var expected = &codeError{errCode: 500}
	if err.Error() != expected.Error() {
		t.Fatalf("Expected %s got %s", expected.Error(), err.Error())
	}
	_, err = tr.RoundTrip(req)
	switch err.(type) {
	case loadshed.Rejected:
	default:
		t.Fatal("Unexpected error type")
	}
}

type fakeLoadShedder struct {
	Counter int32
	err     error
}

func (f *fakeLoadShedder) Do(run func() error) error {
	atomic.AddInt32(&f.Counter, 1)
	if f.err != nil {
		return f.err
	}
	return run()

}
