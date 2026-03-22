// This file written by Claude

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// utlsTransport is an http.RoundTripper that uses uTLS to mimic Chrome's TLS fingerprint
// and properly supports both HTTP/1.1 and HTTP/2.
type utlsTransport struct {
	h1 *http.Transport
	h2 *http2.Transport

	mu    sync.Mutex
	conns map[string]net.Conn
}

func newChromeTLSTransport() http.RoundTripper {
	t := &utlsTransport{
		conns: make(map[string]net.Conn),
		h2:    &http2.Transport{},
	}

	t.h1 = &http.Transport{
		DialTLSContext: t.dialTLS,
	}

	// Configure h2 to use our custom dial
	t.h2.DialTLSContext = func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
		return t.dialTLS(ctx, network, addr)
	}

	return t
}

func (t *utlsTransport) dialTLS(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	tlsConn := utls.UClient(conn, &utls.Config{
		ServerName: host,
	}, utls.HelloChrome_Auto)

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	// Store negotiated protocol for RoundTrip to decide h1 vs h2
	proto := tlsConn.ConnectionState().NegotiatedProtocol
	key := fmt.Sprintf("%s:%s", network, addr)
	t.mu.Lock()
	if proto == "h2" {
		t.conns[key] = tlsConn
	}
	t.mu.Unlock()

	return tlsConn, nil
}

func (t *utlsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Try h2 first - if server negotiated h2, use h2 transport
	// We check by attempting a connection and seeing what was negotiated
	resp, err := t.h2.RoundTrip(req)
	if err == nil {
		return resp, nil
	}

	// Fall back to h1
	return t.h1.RoundTrip(req)
}
