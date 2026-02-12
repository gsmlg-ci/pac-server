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
	customPath  string
)

const defaultGFWListPath = "gfwlist.txt"

//go:embed gfwlist.txt
var embeddedGFWList []byte

func init() {
	flag.StringVar(&host, "h", ":1080", "Set pac server listen address, default is ':1080'.")
	flag.StringVar(&proxyServer, "s", "PROXY 127.0.0.1:3128", "Set proxy server address, default is 'PROXY 127.0.0.1:3128'.")
	flag.BoolVar(&printHosts, "p", false, "Print parsed hosts and exit.")
	flag.StringVar(&gfwlistPath, "g", defaultGFWListPath, "Path to gfwlist.txt (base64 or plain text). If missing and default path is used, embedded gfwlist is used.")
	flag.StringVar(&customPath, "c", "", "Optional path to custom list file.")
}

type pacService struct {
	proxy      string
	gfwlist    string
	custom     string
	mu         sync.RWMutex
	cachedKey  string
	cachedBody []byte
}

func (s *pacService) loadPAC() ([]byte, error) {
	key, err := s.cacheKey()
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	if s.cachedKey == key && len(s.cachedBody) > 0 {
		body := append([]byte(nil), s.cachedBody...)
		s.mu.RUnlock()
		return body, nil
	}
	s.mu.RUnlock()

	domains, err := s.loadDomains()
	if err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return nil, errors.New("no domains parsed from lists")
	}

	pac := []byte(pacgen.GeneratePAC(domains, s.proxy))

	s.mu.Lock()
	s.cachedKey = key
	s.cachedBody = append([]byte(nil), pac...)
	s.mu.Unlock()

	return pac, nil
}

func (s *pacService) loadDomains() ([]string, error) {
	gfwDomains, err := parseDomainsFromFile(s.gfwlist, true)
	if err != nil {
		return nil, err
	}
	if s.custom == "" {
		return gfwDomains, nil
	}

	customDomains, err := parseDomainsFromFile(s.custom, false)
	if err != nil {
		return nil, err
	}

	return pacgen.MergeDomainLists(gfwDomains, customDomains), nil
}

func (s *pacService) cacheKey() (string, error) {
	key, err := sourceCacheKey(s.gfwlist, true)
	if err != nil {
		return "", err
	}
	if s.custom == "" {
		return key, nil
	}

	customKey, err := sourceCacheKey(s.custom, false)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s|%s", key, customKey), nil
}

func sourceCacheKey(path string, allowEmbeddedFallback bool) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if allowEmbeddedFallback && errors.Is(err, os.ErrNotExist) && path == defaultGFWListPath {
			return fmt.Sprintf("g:embedded:%d", len(embeddedGFWList)), nil
		}
		return "", fmt.Errorf("stat %s: %w", path, err)
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
	domains, err := s.loadDomains()
	if err != nil {
		return err
	}
	sort.Strings(domains)
	for _, h := range domains {
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

func main() {
	flag.Parse()

	service := &pacService{
		proxy:   proxyServer,
		gfwlist: gfwlistPath,
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

	log.Printf("PAC server start at %s", host)
	log.Printf("gfwlist source: %s", gfwlistPath)
	if _, err := os.Stat(gfwlistPath); err != nil && errors.Is(err, os.ErrNotExist) && gfwlistPath == defaultGFWListPath {
		log.Printf("gfwlist source file not found, using embedded gfwlist")
	}
	if customPath != "" {
		log.Printf("custom source: %s", customPath)
	}

	log.Fatal(s.ListenAndServe())
}
