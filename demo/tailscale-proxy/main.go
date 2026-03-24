// Demo: Tailscale proxy to a local service (pure Go)
//
// This demonstrates the proxy pattern used by temporal-start-dev,
// running entirely in-process with a test coordination server.
//
// Usage:
//
//	go run demo/tailscale-proxy/main.go
//
// No auth key needed! Uses testcontrol.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	"tailscale.com/tsnet"
	"tailscale.com/tstest/integration/testcontrol"
)

func main() {
	// 1. Start testcontrol server (in-memory coordination server)
	fmt.Println("Starting testcontrol server...")
	control := &testcontrol.Server{}
	controlHTTP := httptest.NewServer(control)
	defer controlHTTP.Close()
	controlURL := controlHTTP.URL
	fmt.Printf("  Control URL: %s\n", controlURL)

	// 2. Start simple HTTP server on localhost:8080
	fmt.Println("Starting HTTP server on localhost:8080...")
	httpLn, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
	defer httpLn.Close()

	go http.Serve(httpLn, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from local server! Request from: %s\n", r.RemoteAddr)
	}))
	fmt.Printf("  Listening on: %s\n", httpLn.Addr())

	// 3. Start tsnet proxy server (hostname: "demo-proxy")
	fmt.Println("Starting proxy (demo-proxy) on tailnet...")
	proxyServer := &tsnet.Server{
		Hostname:   "demo-proxy",
		Dir:        "/tmp/tsnet-demo-proxy",
		ControlURL: controlURL,
	}
	defer proxyServer.Close()

	if err := proxyServer.Start(); err != nil {
		log.Fatalf("Failed to start proxy server: %v", err)
	}

	// Listen on tailnet port 8080
	proxyLn, err := proxyServer.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen on tailnet: %v", err)
	}
	defer proxyLn.Close()

	// Forward connections from tailnet to localhost:8080
	go func() {
		for {
			conn, err := proxyLn.Accept()
			if err != nil {
				return
			}
			go handleProxy(conn, httpLn.Addr().String())
		}
	}()
	fmt.Println("  Proxy listening on tailnet port 8080")

	// Wait for proxy to be ready
	time.Sleep(2 * time.Second)

	// 4. Start tsnet client (hostname: "demo-client")
	fmt.Println("Starting client (demo-client)...")
	client := &tsnet.Server{
		Hostname:   "demo-client",
		Dir:        "/tmp/tsnet-demo-client",
		ControlURL: controlURL,
	}
	defer client.Close()

	if err := client.Start(); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Wait for client to be ready
	time.Sleep(2 * time.Second)

	// 5. Client connects to demo-proxy:8080
	fmt.Println("Client connecting to demo-proxy:8080...")
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(dialCtx context.Context, network, addr string) (net.Conn, error) {
				return client.Dial(dialCtx, network, addr)
			},
		},
	}

	resp, err := httpClient.Get("http://demo-proxy:8080/")
	if err != nil {
		log.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	fmt.Printf("Response: %s", body)
	fmt.Println("\nSuccess! Proxy working correctly.")
}

// handleProxy forwards a connection to the target address
func handleProxy(src net.Conn, targetAddr string) {
	defer src.Close()

	dst, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Failed to dial target: %v", err)
		return
	}
	defer dst.Close()

	// Bidirectional copy
	go io.Copy(dst, src)
	io.Copy(src, dst)
}
