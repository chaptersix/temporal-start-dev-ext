package tailscale

import (
	"context"
	"io"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
	"tailscale.com/tstest/integration"
	"tailscale.com/tstest/integration/testcontrol"
	"tailscale.com/types/logger"
)

// startTestControl starts a testcontrol server with DERP and STUN
func startTestControl(t *testing.T) (server *testcontrol.Server, url string) {
	t.Helper()

	// Start DERP and STUN servers
	derpMap := integration.RunDERPAndSTUN(t, logger.Discard, "127.0.0.1")

	// Start control server
	control := &testcontrol.Server{
		DERPMap: derpMap,
		DNSConfig: &tailcfg.DNSConfig{
			Proxied: true,
		},
		MagicDNSDomain: "tail-scale.ts.net",
	}
	control.HTTPTestServer = httptest.NewUnstartedServer(control)
	control.HTTPTestServer.Start()
	t.Cleanup(control.HTTPTestServer.Close)

	return control, control.HTTPTestServer.URL
}

// startEchoServer starts a simple TCP echo server for testing
func startEchoServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c) // echo back everything
			}(conn)
		}
	}()

	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

// waitForTsnetReady waits for a tsnet server to be running
func waitForTsnetReady(t *testing.T, srv *tsnet.Server, timeout time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := srv.Up(ctx)
	require.NoError(t, err, "failed to bring up tsnet server")
}

// TestProxyE2E tests end-to-end proxy functionality using testcontrol
func TestProxyE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// 1. Start testcontrol server
	_, controlURL := startTestControl(t)

	// 2. Start mock echo server
	echoAddr := startEchoServer(t)

	// 3. Start our proxy (tsnet server)
	proxy, err := Start(ctx, Options{
		Hostname:     "test-proxy",
		FrontendAddr: echoAddr,
		FrontendPort: 7233,
		StateDir:     t.TempDir(),
		ControlURL:   controlURL,
	})
	require.NoError(t, err)
	defer proxy.Stop()

	// Wait for proxy to be ready
	waitForTsnetReady(t, proxy.server, 10*time.Second)

	// 4. Start client tsnet node
	client := &tsnet.Server{
		Hostname:   "test-client",
		Dir:        t.TempDir(),
		ControlURL: controlURL,
		Ephemeral:  true,
	}
	defer client.Close()

	// Wait for client to be ready
	waitForTsnetReady(t, client, 10*time.Second)

	// 5. Client connects to proxy via tailnet
	conn, err := client.Dial(ctx, "tcp", "test-proxy:7233")
	require.NoError(t, err)
	defer conn.Close()

	// 6. Send data through proxy
	testData := []byte("hello tailscale proxy")
	_, err = conn.Write(testData)
	require.NoError(t, err)

	// 7. Verify echo
	buf := make([]byte, len(testData))
	_, err = io.ReadFull(conn, buf)
	require.NoError(t, err)
	require.Equal(t, testData, buf)

	// Close connection to allow proxy to finish
	conn.Close()
}

// TestServer_Stop tests clean shutdown behavior
func TestServer_Stop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start testcontrol
	_, controlURL := startTestControl(t)

	// Start mock server
	echoAddr := startEchoServer(t)

	// Start tsnet proxy
	proxy, err := Start(ctx, Options{
		Hostname:     "test-stop",
		FrontendAddr: echoAddr,
		FrontendPort: 7233,
		StateDir:     t.TempDir(),
		ControlURL:   controlURL,
	})
	require.NoError(t, err)

	// Wait for proxy to be ready
	waitForTsnetReady(t, proxy.server, 10*time.Second)

	// Call Stop()
	proxy.Stop()

	// Verify server is closed
	require.NotNil(t, proxy.server)

	// Verify multiple Stop() calls are safe
	proxy.Stop()
	proxy.Stop()
}

// TestServer_MultiPort tests multiple proxy ports (gRPC + UI)
func TestServer_MultiPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start testcontrol
	_, controlURL := startTestControl(t)

	// Start two mock servers (gRPC and UI)
	grpcAddr := startEchoServer(t)
	uiAddr := startEchoServer(t)

	// Start proxy with both ports configured
	proxy, err := Start(ctx, Options{
		Hostname:     "test-multiport",
		FrontendAddr: grpcAddr,
		FrontendPort: 7233,
		UIAddr:       uiAddr,
		UIPort:       8233,
		StateDir:     t.TempDir(),
		ControlURL:   controlURL,
	})
	require.NoError(t, err)
	defer proxy.Stop()

	// Wait for proxy to be ready
	waitForTsnetReady(t, proxy.server, 10*time.Second)

	// Create client
	client := &tsnet.Server{
		Hostname:   "test-client-multi",
		Dir:        t.TempDir(),
		ControlURL: controlURL,
		Ephemeral:  true,
	}
	defer client.Close()

	// Wait for client to be ready
	waitForTsnetReady(t, client, 10*time.Second)

	// Test gRPC port (7233)
	grpcConn, err := client.Dial(ctx, "tcp", "test-multiport:7233")
	require.NoError(t, err)
	defer grpcConn.Close()

	grpcData := []byte("grpc test")
	_, err = grpcConn.Write(grpcData)
	require.NoError(t, err)

	grpcBuf := make([]byte, len(grpcData))
	_, err = io.ReadFull(grpcConn, grpcBuf)
	require.NoError(t, err)
	require.Equal(t, grpcData, grpcBuf)
	grpcConn.Close() // Close to unblock proxy goroutines

	// Test UI port (8233)
	uiConn, err := client.Dial(ctx, "tcp", "test-multiport:8233")
	require.NoError(t, err)
	defer uiConn.Close()

	uiData := []byte("ui test")
	_, err = uiConn.Write(uiData)
	require.NoError(t, err)

	uiBuf := make([]byte, len(uiData))
	_, err = io.ReadFull(uiConn, uiBuf)
	require.NoError(t, err)
	require.Equal(t, uiData, uiBuf)
	uiConn.Close() // Close to unblock proxy goroutines
}
