package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gsmlg-ci/pac-server/internal/pacgen"
)

var (
	host        string
	proxyServer string
	printHosts  bool
	gfwlistPath string
	domainsPath string
	customPath  string
)

const defaultGFWListPath = "gfwlist.txt"
const defaultDomainsPath = "domains.txt"

//go:embed gfwlist.txt
var embeddedGFWList []byte

func init() {
	flag.StringVar(&host, "h", ":1080", "Set pac server listen address, default is ':1080'.")
	flag.StringVar(&proxyServer, "s", "PROXY 127.0.0.1:3128", "Set proxy server address, default is 'PROXY 127.0.0.1:3128'.")
	flag.BoolVar(&printHosts, "p", false, "Print parsed hosts and exit.")
	flag.StringVar(&gfwlistPath, "g", defaultGFWListPath, "Path to gfwlist.txt (base64 or plain text). If missing and default path is used, embedded gfwlist is used.")
	flag.StringVar(&domainsPath, "d", defaultDomainsPath, "Path to extra domains file (one domain per line). Skipped if file does not exist.")
	flag.StringVar(&customPath, "c", "", "Optional path to custom list file (deprecated, use -d instead).")
}

type pacService struct {
	proxy   string
	gfwlist string
	domains string
	custom  string
	mu      sync.RWMutex
	cached  *cachedPAC
}

type cachedPAC struct {
	key  string
	body []byte
}

func (s *pacService) loadPAC() ([]byte, error) {
	key, err := s.cacheKey()
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	if s.cached != nil && s.cached.key == key {
		body := append([]byte(nil), s.cached.body...)
		s.mu.RUnlock()
		return body, nil
	}
	s.mu.RUnlock()

	customDomains, err := s.loadDomainsFile(s.domains)
	if err != nil {
		return nil, err
	}
	gfwDomains, err := s.loadDomains()
	if err != nil {
		return nil, err
	}

	var allCustom []string
	if len(customDomains) > 0 {
		allCustom = customDomains
	} else if s.custom != "" {
		allCustom, err = parseDomainsFromFile(s.custom, false)
		if err != nil {
			return nil, err
		}
	}

	pac := []byte(pacgen.GeneratePAC(allCustom, gfwDomains, s.proxy))

	s.mu.Lock()
	if s.cached == nil {
		s.cached = &cachedPAC{}
	}
	s.cached.key = key
	s.cached.body = append([]byte(nil), pac...)
	s.mu.Unlock()

	return pac, nil
}

func (s *pacService) loadDomains() ([]string, error) {
	return parseDomainsFromFile(s.gfwlist, true)
}

func (s *pacService) loadDomainsFile(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return pacgen.ParseDomains(string(content)), nil
}

func (s *pacService) cacheKey() (string, error) {
	gfwKey, err := sourceCacheKey(s.gfwlist, true)
	if err != nil {
		return "", err
	}

	domainsKey, err := sourceCacheKey(s.domains, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			domainsKey = ""
		} else {
			return "", err
		}
	}

	customKey, err := sourceCacheKey(s.custom, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			customKey = ""
		} else {
			return "", err
		}
	}

	return fmt.Sprintf("%s|%s|%s", gfwKey, domainsKey, customKey), nil
}

func sourceCacheKey(path string, allowEmbeddedFallback bool) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if allowEmbeddedFallback && errors.Is(err, os.ErrNotExist) && path == defaultGFWListPath {
			return fmt.Sprintf("g:embedded:%d", len(embeddedGFWList)), nil
		}
		return "", err
	}

	return fmt.Sprintf("f:%s:%d:%d", path, stat.ModTime().UnixNano(), stat.Size()), nil
}

func parseDomainsFromFile(path string, allowEmbeddedFallback bool) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if allowEmbeddedFallback && errors.Is(err, os.ErrNotExist) && path == defaultGFWListPath {
			content = embeddedGFWList
		} else {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
	}

	raw, err := pacgen.DecodeMaybeBase64(content)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	return pacgen.ParseDomains(string(raw)), nil
}

func (s *pacService) showHosts() error {
	var domains []string

	if customDomains, err := s.loadDomainsFile(s.domains); err != nil {
		return err
	} else if len(customDomains) > 0 {
		domains = append(domains, customDomains...)
	}

	if customDomains, err := parseDomainsFromFile(s.custom, false); err != nil {
		return err
	} else if len(customDomains) > 0 {
		domains = append(domains, customDomains...)
	}

	if gfwDomains, err := s.loadDomains(); err != nil {
		return err
	} else {
		domains = append(domains, gfwDomains...)
	}

	seen := make(map[string]bool)
	var unique []string
	for _, d := range domains {
		if !seen[d] {
			seen[d] = true
			unique = append(unique, d)
		}
	}

	sort.Strings(unique)
	for _, h := range unique {
		fmt.Println(h)
	}
	return nil
}

func (s *pacService) handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("request from %s", r.RemoteAddr)

	pac, err := s.loadPAC()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate PAC: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pac)))
	_, _ = w.Write(pac)
}

func (s *pacService) watchDomains(done <-chan struct{}) {
	prevModTime := int64(-1)
	if st, err := os.Stat(s.domains); err == nil {
		prevModTime = st.ModTime().UnixNano()
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			st, err := os.Stat(s.domains)
			if err != nil {
				continue
			}
			modTime := st.ModTime().UnixNano()
			if modTime != prevModTime {
				prevModTime = modTime
				s.mu.Lock()
				s.cached = nil
				s.mu.Unlock()
				log.Printf("domains.txt changed, cache invalidated")
			}
		}
	}
}

func main() {
	flag.Parse()

	domainsExist := true
	if _, err := os.Stat(domainsPath); err != nil && errors.Is(err, os.ErrNotExist) {
		domainsExist = false
	}

	service := &pacService{
		proxy:   proxyServer,
		gfwlist: gfwlistPath,
		domains: domainsPath,
		custom:  customPath,
	}

	if printHosts {
		if err := service.showHosts(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	s := &http.Server{
		Addr:           host,
		Handler:        http.HandlerFunc(service.handler),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	done := make(chan struct{})
	defer close(done)

	go service.watchDomains(done)

	log.Printf("PAC server start at %s", host)
	log.Printf("gfwlist source: %s", gfwlistPath)
	if _, err := os.Stat(gfwlistPath); err != nil && errors.Is(err, os.ErrNotExist) && gfwlistPath == defaultGFWListPath {
		log.Printf("gfwlist source file not found, using embedded gfwlist")
	}
	if domainsExist {
		log.Printf("domains source: %s (auto-reload enabled)", domainsPath)
	} else {
		log.Printf("domains source: %s (file not found, skipped)", domainsPath)
	}
	if customPath != "" {
		log.Printf("custom source: %s", customPath)
	}

	log.Fatal(s.ListenAndServe())
}
