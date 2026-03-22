package main

import (
	"net/http"

	"golang.org/x/time/rate"
)

type RateLimitedTransport struct {
	limiter   *rate.Limiter
	transport http.RoundTripper
}

func (t *RateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(req)
}
