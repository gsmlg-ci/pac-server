Runtime flags:

  - `-h`: listen address (default `:1080`)
  - `-s`: proxy string inserted wherever `__PROXY__` appears in the PAC (default `PROXY 127.0.0.1:3128`)
  - `-g`: path to gfwlist.txt (base64 or plain text); if missing and default path is used, embedded gfwlist is used
  - `-d`: path to extra domains file (`domains.txt`), checked before gfwlist (higher priority); auto-reloads on file change, cache invalidates automatically
  - `-n`: path to noproxy domains file (`noproxy.txt`), checked first — matched domains always go DIRECT; auto-reloads on file change
  - `-p`: prints all domains (noproxy + domains.txt + gfwlist) and exits
