# AGENTS.md (pac-server)

This repo is a small Go service that serves a PAC (Proxy Auto-Configuration) file.

## Source Of Truth

- `gfwlist.pac` is generated from upstream GFWList (`gfwlist.txt`, base64 encoded).
- Do not hand-edit `gfwlist.pac` unless you are debugging output. Prefer regenerating it.

## Common Commands

```bash
# Regenerate gfwlist.pac from upstream gfwlist.txt
make update-gfwlist

# Build the server binary
make build

# Run tests
go test ./...
```

## How Updates Work

- Generator: `cmd/gfwlist2pac`
  - Fetches `https://github.com/gfwlist/gfwlist/raw/refs/heads/master/gfwlist.txt`
  - Decodes base64, extracts domain rules, renders `gfwlist.pac`
- Scheduled GitHub Action: `.github/workflows/update-gfwlist.yml`
  - Runs weekly (cron) and commits `gfwlist.pac` changes back to `main`
  - The image build workflow should trigger on `gfwlist.pac` changes

## PAC Server Behavior

- The server embeds `gfwlist.pac` at build time by default.
- Runtime flags in `main.go`:
  - `-s`: proxy string inserted wherever `__PROXY__` appears in the PAC
  - `-pac`: serve a PAC template from disk (overrides embedded PAC)
  - `-custom`: optional JS snippet injected at `/*__CUSTOM_PAC__*/` inside `FindProxyForURL`
  - `-p`: prints domains found in the PAC and exits

## Custom PAC Snippet

- `-custom` injects raw JS inside `FindProxyForURL`.
- Keep the snippet small and deterministic, and return a value when matching:

```js
if (host === "example.com") return direct;
```

## Design Constraints

- Avoid mutating shared global PAC content per request. Treat PAC templates as immutable and render per request.
- Keep generated PAC output stable (deterministic ordering) so diffs are meaningful.

