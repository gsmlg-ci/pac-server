package gfwlist

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// DecodeBase64 decodes gfwlist content which is typically base64-encoded text.
// It tolerates whitespace and newlines in the input.
func DecodeBase64(b []byte) ([]byte, error) {
	clean := bytes.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t', ' ':
			return -1
		default:
			return r
		}
	}, b)

	decoded, err := base64.StdEncoding.DecodeString(string(clean))
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	return decoded, nil
}

var domainRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)

func normalizeDomain(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, ".")
	s = strings.TrimSuffix(s, ".")

	// Strip port if present.
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[:i]
	}

	if s == "" || !strings.Contains(s, ".") {
		return "", false
	}
	if strings.ContainsAny(s, "*_") {
		return "", false
	}
	if !domainRe.MatchString(s) {
		return "", false
	}
	return s, true
}

// ExtractDomains returns two sorted, unique domain lists:
// - proxyDomains: domains that should use proxy
// - directDomains: domains that should go DIRECT (from @@ rules)
//
// This intentionally focuses on domain-based rules, which covers the majority
// of gfwlist entries and yields a fast PAC.
func ExtractDomains(gfwlistText []byte) (proxyDomains []string, directDomains []string) {
	proxySet := map[string]struct{}{}
	directSet := map[string]struct{}{}

	lines := strings.Split(string(gfwlistText), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue
		}

		isDirect := false
		if strings.HasPrefix(line, "@@") {
			isDirect = true
			line = strings.TrimPrefix(line, "@@")
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Drop anchors.
		line = strings.TrimLeft(line, "|")

		// Common autoproxy form "||example.com" (optionally with path).
		line = strings.TrimPrefix(line, "||")

		// If it's a full URL, parse hostname.
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			u, err := url.Parse(line)
			if err != nil {
				continue
			}
			if d, ok := normalizeDomain(u.Hostname()); ok {
				if isDirect {
					directSet[d] = struct{}{}
				} else {
					proxySet[d] = struct{}{}
				}
			}
			continue
		}

		// Cut off path/query.
		if i := strings.IndexAny(line, "/?"); i >= 0 {
			line = line[:i]
		}

		// Ignore patterns requiring full URL matching for now.
		if strings.ContainsAny(line, "*%") {
			continue
		}

		// Remove leading dot.
		line = strings.TrimPrefix(line, ".")

		if d, ok := normalizeDomain(line); ok {
			if isDirect {
				directSet[d] = struct{}{}
			} else {
				proxySet[d] = struct{}{}
			}
		}
	}

	for d := range proxySet {
		proxyDomains = append(proxyDomains, d)
	}
	for d := range directSet {
		directDomains = append(directDomains, d)
	}
	sort.Strings(proxyDomains)
	sort.Strings(directDomains)
	return proxyDomains, directDomains
}
