# Tailscale Proxy Demo

## Pure Go Demo (No External Services)

This demo shows the complete proxy pattern in a single process:

```bash
go run demo/tailscale-proxy/main.go
```

**What it does**:
- Runs a test Tailscale coordination server in-memory
- Creates a proxy that forwards traffic from tailnet to localhost
- Creates a client that connects through the proxy
- Verifies end-to-end connectivity

**No prerequisites needed!** Everything runs in-process.

## How It Works

The proxy pattern:
1. Local service listens on 127.0.0.1:8080 (simulated HTTP server)
2. testcontrol.Server provides coordination (replaces login.tailscale.com)
3. tsnet proxy joins the tailnet with hostname "demo-proxy"
4. tsnet client joins the tailnet with hostname "demo-client"
5. Client connects to demo-proxy:8080 → proxied to localhost:8080

This is exactly what `temporal start-dev --tailscale` does, but for testing we use testcontrol instead of real Tailscale infrastructure.

## Using with Real Tailscale (Optional)

To use a real tailnet instead of testcontrol:
```go
// Replace testcontrol setup with:
export TS_AUTHKEY=tskey-auth-...
// Remove ControlURL from tsnet.Server configs
```

The proxy pattern remains the same, just connected to production Tailscale instead of the test control server.
