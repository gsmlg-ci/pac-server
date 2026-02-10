package pac

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type Lists struct {
	ProxyDomains  []string
	DirectDomains []string
}

func RenderPAC(lists Lists) ([]byte, error) {
	proxy := uniqSorted(lists.ProxyDomains)
	direct := uniqSorted(lists.DirectDomains)

	var b bytes.Buffer
	_, _ = b.WriteString("// Generated from gfwlist.txt\n")
	_, _ = b.WriteString("// Proxy placeholder: __PROXY__\n\n")
	_, _ = b.WriteString("var proxy = \"__PROXY__\";\n")
	_, _ = b.WriteString("var direct = \"DIRECT\";\n\n")

	_, _ = b.WriteString("var proxyDomains = {\n")
	writeDomainMap(&b, proxy)
	_, _ = b.WriteString("};\n\n")

	_, _ = b.WriteString("var directDomains = {\n")
	writeDomainMap(&b, direct)
	_, _ = b.WriteString("};\n\n")

	_, _ = b.WriteString("function matchDomain(map, host) {\n")
	_, _ = b.WriteString("    if (!host) return false;\n")
	_, _ = b.WriteString("    host = host.toLowerCase();\n")
	_, _ = b.WriteString("    if (map[host] === 1) return true;\n")
	_, _ = b.WriteString("    var pos = host.indexOf('.');\n")
	_, _ = b.WriteString("    while (pos !== -1) {\n")
	_, _ = b.WriteString("        host = host.substring(pos + 1);\n")
	_, _ = b.WriteString("        if (map[host] === 1) return true;\n")
	_, _ = b.WriteString("        pos = host.indexOf('.');\n")
	_, _ = b.WriteString("    }\n")
	_, _ = b.WriteString("    return false;\n")
	_, _ = b.WriteString("}\n\n")

	_, _ = b.WriteString("function FindProxyForURL(url, host) {\n")
	_, _ = b.WriteString("    /*__CUSTOM_PAC__*/\n")
	_, _ = b.WriteString("    if (matchDomain(directDomains, host)) return direct;\n")
	_, _ = b.WriteString("    if (matchDomain(proxyDomains, host)) return proxy;\n")
	_, _ = b.WriteString("    return direct;\n")
	_, _ = b.WriteString("}\n")

	out := b.Bytes()
	if !bytes.Contains(out, []byte("/*__CUSTOM_PAC__*/")) {
		return nil, fmt.Errorf("missing custom placeholder")
	}
	return out, nil
}

func uniqSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	cp := append([]string(nil), in...)
	for i := range cp {
		cp[i] = strings.ToLower(strings.TrimSpace(cp[i]))
	}
	sort.Strings(cp)
	out := make([]string, 0, len(cp))
	var last string
	for _, s := range cp {
		if s == "" || s == last {
			continue
		}
		out = append(out, s)
		last = s
	}
	return out
}

func writeDomainMap(b *bytes.Buffer, domains []string) {
	for i, d := range domains {
		// Keep stable output and avoid trailing commas (older JS engines).
		if i == len(domains)-1 {
			_, _ = fmt.Fprintf(b, "    \"%s\": 1\n", d)
		} else {
			_, _ = fmt.Fprintf(b, "    \"%s\": 1,\n", d)
		}
	}
}
