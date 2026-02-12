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
	pac := GeneratePAC([]string{"example.com"}, "PROXY 127.0.0.1:3128")

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
