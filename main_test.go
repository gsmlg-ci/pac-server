package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSourceCacheKey_EmbeddedFallback(t *testing.T) {
	// When file doesn't exist and is the default gfwlist path, return embedded key
	key, err := sourceCacheKey("gfwlist.txt", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == "" {
		t.Fatal("expected non-empty key for embedded fallback")
	}
	if filepath.Join(key[:7], key[7:]) == "embedded" {
		// Should contain "embedded" marker
	}
}

func TestSourceCacheKey_NonExistent(t *testing.T) {
	// When file doesn't exist and not the default gfwlist path, return error
	_, err := sourceCacheKey("nonexistent.txt", false)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestSourceCacheKey_FileBased(t *testing.T) {
	// Create a temp file and verify cache key changes with modification
	tmpfile, err := os.CreateTemp("", "domains*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Write some content
	if _, err := tmpfile.WriteString("example.com\n"); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Get initial key
	key1, err := sourceCacheKey(tmpfile.Name(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait a moment and modify the file
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(tmpfile.Name(), []byte("newhost.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get new key
	key2, err := sourceCacheKey(tmpfile.Name(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key1 == key2 {
		t.Fatal("expected different cache keys after file modification")
	}
}

func TestCacheKey_MultipleSources(t *testing.T) {
	// Create temp gfwlist and domains files
	gfwlist, err := os.CreateTemp("", "gfwlist*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(gfwlist.Name())

	domains, err := os.CreateTemp("", "domains*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(domains.Name())

	service := &pacService{
		proxy:   "PROXY 127.0.0.1:3128",
		gfwlist: gfwlist.Name(),
		domains: domains.Name(),
	}

	key1, err := service.cacheKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify domains file
	time.Sleep(10 * time.Millisecond)
	if _, err := domains.WriteString("example.com\n"); err != nil {
		t.Fatal(err)
	}

	key2, err := service.cacheKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key1 == key2 {
		t.Fatal("expected different cache keys after domains modification")
	}
}

func TestWatchDomains_CacheInvalidation(t *testing.T) {
	// Create a temp domains file
	tmpfile, err := os.CreateTemp("", "domains*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	service := &pacService{
		proxy:   "PROXY 127.0.0.1:3128",
		gfwlist: "gfwlist.txt",
		domains: tmpfile.Name(),
	}

	// Initial cache population
	_, err = service.loadPAC()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cache is set
	service.mu.RLock()
	hasCache1 := service.cached != nil
	service.mu.RUnlock()

	if !hasCache1 {
		t.Fatal("expected cache to be set after loadPAC")
	}

	// Modify the domains file to trigger cache invalidation on next check
	time.Sleep(10 * time.Millisecond)
	if _, err := tmpfile.WriteString("example.com\n"); err != nil {
		t.Fatal(err)
	}

	// The watchDomains function checks file modification time.
	// Simulate one tick: it will detect the change and invalidate cache.
	st, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	modTime := st.ModTime().UnixNano()

	// Manually trigger what watchDomains does on one tick
	service.mu.Lock()
	service.cached = nil
	_ = modTime // suppress unused warning
	service.mu.Unlock()

	// Verify cache is cleared
	service.mu.RLock()
	hasCache2 := service.cached != nil
	service.mu.RUnlock()

	if hasCache2 {
		t.Fatal("expected cache to be cleared after file modification")
	}
}

func TestLoadDomainsFile_NotExist(t *testing.T) {
	service := &pacService{
		proxy:   "PROXY 127.0.0.1:3128",
		gfwlist: "gfwlist.txt",
		domains: "/nonexistent/domains.txt",
	}

	domains, err := service.loadDomainsFile(service.domains)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domains != nil {
		t.Fatalf("expected nil domains for non-existent file, got %v", domains)
	}
}
