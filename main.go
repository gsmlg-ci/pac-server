package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	host        string
	proxyServer string
	printHosts  bool
	pacPath     string
	customPath  string
)

func init() {
	flag.StringVar(&host, "h", ":1080", "Set pac server listen address, default is ':1080'.")
	flag.StringVar(&proxyServer, "s", "PROXY 127.0.0.1:3128", "Set proxy server address, default is 'PROXY 127.0.0.1:3128'.")
	flag.BoolVar(&printHosts, "p", false, "Print hosts in gfwlist.pac.")
	flag.StringVar(&pacPath, "pac", "", "Serve PAC content from a file path (defaults to embedded gfwlist.pac).")
	flag.StringVar(&customPath, "custom", "", "Optional custom PAC snippet file to inject at /*__CUSTOM_PAC__*/.")
}

//go:embed gfwlist.pac
var embeddedPAC []byte

func pacHandler(pacTemplate []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request from %s %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		pac := string(pacTemplate)
		pac = strings.ReplaceAll(pac, "__PROXY__", proxyServer)

		if customPath != "" {
			b, err := os.ReadFile(customPath)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to read custom PAC: %v", err), http.StatusInternalServerError)
				return
			}
			pac = strings.Replace(pac, "/*__CUSTOM_PAC__*/", string(b), 1)
		} else {
			pac = strings.Replace(pac, "/*__CUSTOM_PAC__*/", "", 1)
		}

		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pac)))

		_, _ = w.Write([]byte(pac))
	}
}

func showHosts(pacTemplate []byte) {
	// Extract domains from the generated JS maps:
	// var proxyDomains = {"example.com":1,...};
	// var directDomains = {"example.org":1,...};
	pac := string(pacTemplate)
	pac = strings.ReplaceAll(pac, "__PROXY__", proxyServer)
	pac = strings.Replace(pac, "/*__CUSTOM_PAC__*/", "", 1)

	re, _ := regexp.Compile("\"([a-z0-9.-]+)\":\\s*1")
	m := re.FindAllStringSubmatch(pac, -1)
	seen := map[string]struct{}{}
	for _, mm := range m {
		if len(mm) < 2 {
			continue
		}
		seen[mm[1]] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	sort.Strings(out)
	for _, h := range out {
		fmt.Println(h)
	}
}

func loadPACTemplate() ([]byte, error) {
	if pacPath == "" {
		return embeddedPAC, nil
	}
	return os.ReadFile(pacPath)
}

func main() {
	flag.Parse()

	pacTemplate, err := loadPACTemplate()
	if err != nil {
		log.Fatalf("failed to load PAC template: %v", err)
	}

	if printHosts {
		showHosts(pacTemplate)
		os.Exit(0)
	}

	s := &http.Server{
		Addr:           host,
		Handler:        pacHandler(pacTemplate),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("PAC Server start at %s", host)

	log.Fatal(s.ListenAndServe())
}
