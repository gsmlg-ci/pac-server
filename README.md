# pac-server

[![Build](https://github.com/gsmlg-ci/pac-server/actions/workflows/build.yml/badge.svg)](https://github.com/gsmlg-ci/pac-server/actions/workflows/build.yml)

A simple PAC (Proxy Auto-Configuration) server written in Go. Serves a GFW list PAC file with configurable proxy settings.

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
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-h` | `:1080` | Listen address |
| `-s` | `PROXY 127.0.0.1:3128` | Proxy server address |
| `-p` | `false` | Print hosts in gfwlist.pac and exit |

## Build

```bash
# Download latest gfwlist.pac and build binary
make download && make build

# Build Docker image
docker build -t gsmlg/pac-server .
```
