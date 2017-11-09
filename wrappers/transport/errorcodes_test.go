package transport

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestNoErrorCode(t *testing.T) {
	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	var tr = NewErrCodes([]int{})(wrapped)
	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))

	var _, err = tr.RoundTrip(req)
	if err != nil {
		t.Fatal("Got unexpected error")
	}
}

func TestDoesNotMatchErrorCode(t *testing.T) {
	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	var tr = NewErrCodes([]int{500, 501, 502, 503})(wrapped)
	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))

	var _, err = tr.RoundTrip(req)
	if err != nil {
		t.Fatal("Got unexpected error")
	}
}

func TestMatchesErrorCode(t *testing.T) {
	resp := &http.Response{
		Status:     "OK",
		StatusCode: 500,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	var tr = NewErrCodes([]int{500, 501, 502, 503})(wrapped)
	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))

	var _, err = tr.RoundTrip(req)
	if err == nil {
		t.Fatal("Expected error got nil")
	}
}
