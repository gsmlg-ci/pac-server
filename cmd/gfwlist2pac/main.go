package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gsmlg-ci/pac-server/internal/pacgen"
)

const defaultGFWListURL = "https://raw.githubusercontent.com/gfwlist/gfwlist/refs/heads/master/gfwlist.txt"

func main() {
	urlFlag := flag.String("url", defaultGFWListURL, "gfwlist source URL")
	inFlag := flag.String("in", "", "optional local gfwlist.txt path (base64 encoded); use '-' for stdin")
	outFlag := flag.String("out", "gfwlist.pac", "output PAC file path")
	proxyFlag := flag.String("s", pacgen.DefaultProxy, "proxy server value in PAC")
	flag.Parse()

	data, err := readInput(*inFlag, *urlFlag)
	if err != nil {
		fail(err)
	}

	raw, err := pacgen.DecodeMaybeBase64(data)
	if err != nil {
		fail(fmt.Errorf("decode gfwlist: %w", err))
	}

	domains := pacgen.ParseDomains(string(raw))
	if len(domains) == 0 {
		fail(errors.New("no domains parsed from gfwlist"))
	}

	pac := pacgen.GeneratePAC(domains, *proxyFlag)
	if err := os.WriteFile(*outFlag, []byte(pac), 0o644); err != nil {
		fail(fmt.Errorf("write PAC file: %w", err))
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func readInput(inPath, url string) ([]byte, error) {
	switch inPath {
	case "":
		return download(url)
	case "-":
		return io.ReadAll(os.Stdin)
	default:
		return os.ReadFile(inPath)
	}
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}
