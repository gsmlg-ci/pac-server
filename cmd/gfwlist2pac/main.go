package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gsmlg-ci/pac-server/internal/gfwlist"
	"github.com/gsmlg-ci/pac-server/internal/pac"
)

func main() {
	var (
		inPath  string
		outPath string
		urlStr  string
		timeout time.Duration
	)

	flag.StringVar(&urlStr, "url", "https://github.com/gfwlist/gfwlist/raw/refs/heads/master/gfwlist.txt", "GFWList base64 URL")
	flag.StringVar(&inPath, "in", "", "Read gfwlist base64 from a local file instead of -url")
	flag.StringVar(&outPath, "out", "gfwlist.pac", "Write PAC output path")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "HTTP request timeout")
	flag.Parse()

	var raw []byte
	var err error

	if inPath != "" {
		raw, err = os.ReadFile(inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read -in: %v\n", err)
			os.Exit(2)
		}
	} else {
		c := &http.Client{Timeout: timeout}
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "new request: %v\n", err)
			os.Exit(2)
		}
		req.Header.Set("User-Agent", "pac-server gfwlist2pac")
		resp, err := c.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch -url: %v\n", err)
			os.Exit(2)
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			fmt.Fprintf(os.Stderr, "fetch -url: unexpected status %s\n", resp.Status)
			os.Exit(2)
		}
		raw, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read response: %v\n", err)
			os.Exit(2)
		}
	}

	decoded, err := gfwlist.DecodeBase64(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode gfwlist: %v\n", err)
		os.Exit(2)
	}

	proxyDomains, directDomains := gfwlist.ExtractDomains(decoded)
	pacBytes, err := pac.RenderPAC(pac.Lists{
		ProxyDomains:  proxyDomains,
		DirectDomains: directDomains,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "render pac: %v\n", err)
		os.Exit(2)
	}

	if err := os.WriteFile(outPath, pacBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write -out: %v\n", err)
		os.Exit(2)
	}
}
