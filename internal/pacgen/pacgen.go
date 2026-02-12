package pacgen

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"
)

const DefaultProxy = "SOCKS5 127.0.0.1:1080; SOCKS 127.0.0.1:1080; DIRECT;"

var domainPattern = regexp.MustCompile(`(?i)[a-z0-9][a-z0-9.-]*\.[a-z0-9-]{2,}`)

func DecodeMaybeBase64(input []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(input))
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		if strings.Contains(trimmed, "\n") || strings.Contains(trimmed, "||") {
			return []byte(trimmed), nil
		}
		return nil, err
	}
	return decoded, nil
}

func ParseDomains(raw string) []string {
	set := make(map[string]struct{})
	s := bufio.NewScanner(strings.NewReader(raw))

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			continue
		}
		if strings.HasPrefix(line, "/") && strings.HasSuffix(line, "/") {
			continue
		}

		for _, d := range domainPattern.FindAllString(line, -1) {
			normalized, ok := normalizeDomain(d)
			if ok {
				set[normalized] = struct{}{}
			}
		}
	}

	return SortedDomains(set)
}

func SortedDomains(set map[string]struct{}) []string {
	domains := make([]string, 0, len(set))
	for d := range set {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	return domains
}

func MergeDomainLists(lists ...[]string) []string {
	set := make(map[string]struct{})
	for _, domains := range lists {
		for _, d := range domains {
			if normalized, ok := normalizeDomain(d); ok {
				set[normalized] = struct{}{}
			}
		}
	}
	return SortedDomains(set)
}

func GeneratePAC(domains []string, proxy string) string {
	if proxy == "" {
		proxy = DefaultProxy
	}

	var b strings.Builder
	b.Grow(1024 + len(domains)*24)

	b.WriteString("var proxy = '")
	b.WriteString(proxy)
	b.WriteString("';\n")
	b.WriteString("var hosts = [\n")
	for _, d := range domains {
		fmt.Fprintf(&b, "            %q,\n", d)
	}
	b.WriteString("];\n\n")
	b.WriteString("function FindProxyForURL(url, host) {\n")
	b.WriteString("    var h = host.toLowerCase();\n")
	b.WriteString("    for (var i = 0; i < hosts.length; i++) {\n")
	b.WriteString("        var d = hosts[i];\n")
	b.WriteString("        if (h === d || h.endsWith('.' + d)) {\n")
	b.WriteString("            return proxy;\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n")
	b.WriteString("    return 'DIRECT';\n")
	b.WriteString("}\n")

	return b.String()
}

func normalizeDomain(in string) (string, bool) {
	d := strings.Trim(strings.ToLower(in), ".")
	d = strings.TrimPrefix(d, "*.")
	if d == "" || strings.Contains(d, "*") || !strings.Contains(d, ".") {
		return "", false
	}
	if net.ParseIP(d) != nil {
		return "", false
	}
	if strings.Contains(d, "..") {
		return "", false
	}

	parts := strings.Split(d, ".")
	for _, p := range parts {
		if p == "" || strings.HasPrefix(p, "-") || strings.HasSuffix(p, "-") {
			return "", false
		}
		for _, c := range p {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				return "", false
			}
		}
	}

	return d, true
}
