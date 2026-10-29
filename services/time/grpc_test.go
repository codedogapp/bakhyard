package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
)

// grpcClient returns a client that calls router, served until the test ends, with the gRPC
// protocol.
func grpcClient(t *testing.T, router http.Handler) *connect.Client {
	t.Helper()

	// gRPC needs HTTP/2; between services it runs without TLS (h2c), as internal/server allows.
	h2c := new(http.Protocols)
	h2c.SetUnencryptedHTTP2(true)

	server := httptest.NewUnstartedServer(router)
	server.Config.Protocols = h2c
	server.Start()
	t.Cleanup(server.Close)

	transport := &http.Transport{Protocols: h2c}
	t.Cleanup(transport.CloseIdleConnections)

	httpClient := &http.Client{Transport: transport}

	return connect.NewClient(
		connecthttp.NewTransport(httpClient, server.URL, connecthttp.WithGRPC()),
	)
}
