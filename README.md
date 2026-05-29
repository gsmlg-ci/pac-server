# pac-server

[![Build](https://github.com/gsmlg-ci/pac-server/actions/workflows/build.yml/badge.svg)](https://github.com/gsmlg-ci/pac-server/actions/workflows/build.yml)

A simple PAC (Proxy Auto-Configuration) server written in Go. It embeds `gfwlist.txt` into the binary and generates PAC content at runtime, with optional overrides from external files.

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

# Run with custom domain lists
docker run -d -p 1080:1080 \
  -v $(pwd)/domains.txt:/data/domains.txt:ro \
  -v $(pwd)/noproxy.txt:/data/noproxy.txt:ro \
  gsmlg/pac-server:latest -d /data/domains.txt -n /data/noproxy.txt
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-h` | `:1080` | Listen address |
| `-s` | `PROXY 127.0.0.1:3128` | Proxy server address |
| `-g` | `gfwlist.txt` | Path to gfwlist source file (base64 or plain text). Falls back to embedded list when default file is missing |
| `-d` | `domains.txt` | Path to extra proxy domains file (one domain per line). Skipped if file does not exist |
| `-n` | `noproxy.txt` | Path to noproxy domains file (one domain per line). Matched domains always go DIRECT. Skipped if file does not exist |
| `-c` | `` | Optional path to custom domain list file (deprecated, use `-d` instead) |
| `-p` | `false` | Print parsed hosts and exit |

### Domain Files

Both `domains.txt` and `noproxy.txt` use the same format — one domain per line:

```
example.com
sub.example.org
```

- `example.com` matches `example.com` and all subdomains (e.g. `www.example.com`)
- Lines starting with `!` or `[` are treated as comments and ignored

#### TLD Matching

You can match entire TLDs by prefixing with a dot:

```
.ai
.dev
```

- `.ai` matches **all** `.ai` domains (e.g. `x.ai`, `foo.bar.ai`)

#### Evaluation Order

1. **noproxy.txt** is checked first — matched domains always return `DIRECT`
2. **domains.txt** (custom proxy domains) is checked next
3. **gfwlist** domains are checked last
4. Everything else returns `DIRECT`

Both `domains.txt` and `noproxy.txt` support **auto-reload** — changes are picked up automatically within a few seconds without restarting the server.

## Build

```bash
# Download latest gfwlist.txt and build binary
make download && make build

# Build Docker image
docker build -t gsmlg/pac-server .
```

> **Note:** `gfwlist.pac` is generated from upstream gfwlist — do not hand-edit it.
> Run `make update-gfwlist` to regenerate.

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
