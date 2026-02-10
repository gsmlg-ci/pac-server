package gfwlist

import (
	"encoding/base64"
	"testing"
)

func TestDecodeBase64(t *testing.T) {
	plain := []byte("! comment\n||example.com\n@@||direct.example.com\n")
	enc := []byte(base64.StdEncoding.EncodeToString(plain))

	dec, err := DecodeBase64(enc)
	if err != nil {
		t.Fatalf("DecodeBase64: %v", err)
	}
	if string(dec) != string(plain) {
		t.Fatalf("decoded mismatch: got %q want %q", string(dec), string(plain))
	}
}

func TestExtractDomains(t *testing.T) {
	plain := []byte(`
! comment
||Example.COM
@@||DIRECT.example.com
|https://Sub.URL.example.net/path
.dotprefix.example.org
not.a.domain
`)
	proxy, direct := ExtractDomains(plain)

	if len(proxy) == 0 || len(direct) == 0 {
		t.Fatalf("expected proxy and direct domains, got proxy=%v direct=%v", proxy, direct)
	}
}
