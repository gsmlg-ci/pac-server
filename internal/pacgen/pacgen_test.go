package pacgen

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecodeMaybeBase64(t *testing.T) {
	raw := "||example.com\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))

	decoded, err := DecodeMaybeBase64([]byte(encoded))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(decoded) != raw {
		t.Fatalf("decoded content mismatch\nwant: %q\n got: %q", raw, string(decoded))
	}
}

func TestParseDomains(t *testing.T) {
	raw := strings.Join([]string{
		"!comment",
		"[AutoProxy 0.2.9]",
		"@@||google.com",
		"||youtube.com",
		"|https://www.example.org/path",
		"plain-text with github.com and duplicate github.com",
		"/regex-rule/",
		"127.0.0.1",
	}, "\n")

	domains := ParseDomains(raw)
	got := strings.Join(domains, ",")
	want := "github.com,www.example.org,youtube.com"
	if got != want {
		t.Fatalf("domains mismatch\nwant: %s\n got: %s", want, got)
	}
}

func TestMergeDomainLists(t *testing.T) {
	merged := MergeDomainLists(
		[]string{"Example.com", "a.example.com"},
		[]string{"example.com", "github.com", "127.0.0.1"},
	)

	got := strings.Join(merged, ",")
	want := "a.example.com,example.com,github.com"
	if got != want {
		t.Fatalf("domains mismatch\nwant: %s\n got: %s", want, got)
	}
}

func TestGeneratePAC(t *testing.T) {
	pac := GeneratePAC(nil, nil, []string{"example.com"}, "PROXY 127.0.0.1:3128")

	checks := []string{
		"var proxy = 'PROXY 127.0.0.1:3128';",
		"\"example.com\",",
		"if (h === d || h.endsWith('.' + d))",
		"return 'DIRECT';",
	}

	for _, c := range checks {
		if !strings.Contains(pac, c) {
			t.Fatalf("generated PAC missing expected content: %q", c)
		}
	}
}

func TestGeneratePACWithCustom(t *testing.T) {
	custom := []string{"custom.example.com"}
	gfwlist := []string{"gfwlist.example.com"}

	pac := GeneratePAC(nil, custom, gfwlist, "PROXY 127.0.0.1:3128")

	checks := []string{
		"var customHosts = [",
		"\"custom.example.com\",",
		"\"gfwlist.example.com\",",
		"customHosts.length",
	}

	for _, c := range checks {
		if !strings.Contains(pac, c) {
			t.Fatalf("generated PAC missing expected content: %q", c)
		}
	}

	// Custom domains should appear before gfwlist domains in the file.
	customIdx := strings.Index(pac, "customHosts")
	gfwlistIdx := strings.Index(pac, "var hosts")
	if customIdx > gfwlistIdx {
		t.Fatal("customHosts should appear before hosts in generated PAC")
	}
}

func TestParseDomainsTLD(t *testing.T) {
	raw := strings.Join([]string{
		".ai",
		".dev",
		"example.com",
		".invalid!",
		".-bad",
	}, "\n")

	domains := ParseDomains(raw)
	got := strings.Join(domains, ",")
	want := "ai,dev,example.com"
	if got != want {
		t.Fatalf("domains mismatch\nwant: %s\n got: %s", want, got)
	}
}

func TestGeneratePACWithTLD(t *testing.T) {
	// "ai" as a TLD entry should match any .ai domain
	pac := GeneratePAC(nil, []string{"ai"}, nil, "PROXY 127.0.0.1:3128")

	// The endsWith check: h.endsWith('.' + d) where d="ai" → h.endsWith('.ai')
	if !strings.Contains(pac, `"ai"`) {
		t.Fatal("generated PAC should contain the TLD entry \"ai\"")
	}
	if !strings.Contains(pac, "h.endsWith('.' + d)") {
		t.Fatal("generated PAC should use endsWith matching")
	}
}

func TestGeneratePACWithNoProxy(t *testing.T) {
	noproxy := []string{"internal.example.com"}
	gfwlist := []string{"example.com"}

	pac := GeneratePAC(noproxy, nil, gfwlist, "PROXY 127.0.0.1:3128")

	checks := []string{
		"var noProxyHosts = [",
		"\"internal.example.com\",",
		"return 'DIRECT';",
		"\"example.com\",",
	}

	for _, c := range checks {
		if !strings.Contains(pac, c) {
			t.Fatalf("generated PAC missing expected content: %q", c)
		}
	}

	// noProxyHosts should appear before hosts in the generated PAC.
	noproxyIdx := strings.Index(pac, "noProxyHosts")
	hostsIdx := strings.Index(pac, "var hosts")
	if noproxyIdx > hostsIdx {
		t.Fatal("noProxyHosts should appear before hosts in generated PAC")
	}

	// The DIRECT return for noproxy should come before the proxy return.
	directIdx := strings.Index(pac, "return 'DIRECT';")
	proxyIdx := strings.Index(pac, "return proxy;")
	if directIdx > proxyIdx {
		t.Fatal("noproxy DIRECT return should appear before proxy return")
	}
}
