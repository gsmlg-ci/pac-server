# pac-server

[![Build](https://github.com/gsmlg-ci/pac-server/actions/workflows/build.yml/badge.svg)](https://github.com/gsmlg-ci/pac-server/actions/workflows/build.yml)

A simple PAC (Proxy Auto-Configuration) server written in Go. It embeds `gfwlist.txt` into the binary and generates PAC content at runtime, with optional overrides from external `gfwlist.txt` and `custom.txt`.

Docker image published to:
- `docker.io/gsmlg/pac-server`
- `ghcr.io/gsmlg-dev/pac-server`

## Usage

```bash
# From Docker Hub
docker pull gsmlg/pac-server:latest

# From GitHub Container Registry
docker pull ghcr.io/gsmlg-dev/pac-server:latest

# Run with default settings (listens on :1080, proxy PROXY 127.0.0.1:3128)
docker run -d -p 1080:1080 gsmlg/pac-server:latest

# Run with custom proxy
docker run -d -p 1080:1080 gsmlg/pac-server:latest -s "SOCKS5 127.0.0.1:1080" -h ":1080"

# Run with custom list merged into gfwlist
docker run -d -p 1080:1080 -v $(pwd)/custom.txt:/data/custom.txt:ro gsmlg/pac-server:latest -c /data/custom.txt
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-h` | `:1080` | Listen address |
| `-s` | `PROXY 127.0.0.1:3128` | Proxy server address |
| `-g` | `gfwlist.txt` | Path to gfwlist source file (base64 or plain text). Falls back to embedded list when default file is missing |
| `-c` | `` | Optional path to custom domain list file |
| `-p` | `false` | Print parsed hosts and exit |

## Build

```bash
# Download latest gfwlist.txt and build binary
make download && make build

# Build Docker image
docker build -t gsmlg/pac-server .
```

## Release Artifacts

Tagging `v*` (for example `v1.2.3`) triggers release automation that:
- builds and uploads binaries for:
  - linux amd64/arm64
  - macOS amd64/arm64
  - windows amd64/arm64
  - freebsd amd64/arm64
- publishes multi-arch Docker images (`linux/amd64`, `linux/arm64`) to:
  - `docker.io/gsmlg/pac-server:<tag>` and `:latest`
  - `ghcr.io/gsmlg-dev/pac-server:<tag>` and `:latest`
