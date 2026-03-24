# temporal-ts-net

Extension command for Temporal CLI that exposes the Temporal development server on your Tailscale tailnet.

This extension provides:

- `temporal ts-net` as an enhanced wrapper for `temporal server start-dev`
- Tailscale networking integration to expose your dev server across your tailnet
- Simple setup with automatic Tailscale authentication

## Install

Build the extension binary:

```bash
# Using mage (if installed)
mage build

# Using go run (no mage installation required)
go run mage.go build

# Or directly with go
go build -o ./bin/temporal-ts_net ./cmd/temporal-ts_net
```

Add `./bin` to your `PATH` and verify discovery:

```bash
temporal help --all
```

You should see `ts-net` listed as an extension command.

## Usage

Start local dev server without Tailscale:

```bash
temporal ts-net
```

Expose dev server on Tailscale tailnet:

```bash
temporal ts-net \
    --tailscale \
    --tailscale-hostname your-dev-host
```

`--tsnet` and related `--tsnet-*` flags are also accepted aliases.

Pass any `temporal server start-dev` flags through directly:

```bash
temporal ts-net \
    --tailscale \
    --port 7234 \
    --ui-port 8234 \
    --db-filename /tmp/temporal-dev.db
```

## Extension flags

- `--tailscale` / `--tsnet`: enable tsnet listener and proxy
- `--tailscale-hostname` / `--tsnet-hostname`: tsnet hostname (default `temporal-dev`)
- `--tailscale-authkey` / `--tsnet-authkey`: auth key for non-interactive auth (or set `TS_AUTHKEY` env var)
- `--tailscale-state-dir` / `--tsnet-state-dir`: local state dir for tsnet node

All non-extension flags are forwarded to `temporal server start-dev`.

## Testing

### Running Tests

All tests (including Tailscale):
```bash
go test ./...
```

Tailscale tests run entirely in-process using testcontrol - no external services or auth keys needed!

Verbose output:
```bash
go test ./internal/tailscale -v
```

### Demo

See the [demo](demo/) directory for a self-contained example of the Tailscale proxy pattern:

```bash
go run demo/tailscale-proxy/main.go
```

This demonstrates the complete proxy flow used by `temporal ts-net --tailscale`, running entirely in-process with no external dependencies.

### CI Testing

Tailscale tests run automatically in CI with no configuration needed. They use testcontrol for isolated, reproducible testing.

## Development

This project uses [Mage](https://magefile.org/) for build tasks. You can use mage directly or via `go run` (zero-install):

```bash
# List all available targets
mage -l
# or (no mage installation required)
go run mage.go -l

# Available targets:
go run mage.go build    # Build the binary
go run mage.go test     # Run tests
go run mage.go fmt      # Format code
go run mage.go clean    # Remove build artifacts
go run mage.go install  # Install to $GOPATH/bin
```

## Releases

This project uses [GoReleaser](https://goreleaser.com/) with GitHub Actions to automate releases.

To create a new release:

```bash
# Tag the commit
git tag -a v0.1.0 -m "Release v0.1.0"

# Push the tag
git push origin v0.1.0
```

GitHub Actions will automatically:
- Build binaries for Linux, macOS, and Windows (amd64 and arm64)
- Create a GitHub release with changelog
- Upload release artifacts and checksums

Download pre-built binaries from the [Releases](https://github.com/chaptersix/temporal-ts-net/releases) page.
