package pac

import (
	"fmt"
	"strings"
)

func Generate(domains []string, proxyAddr string) string {
	var sb strings.Builder
	sb.WriteString("function FindProxyForURL(url, host) {\n")
	for _, d := range domains {
		fmt.Fprintf(&sb, "	if (dnsDomainIs(host, %q)) return \"PROXY %s\";\n", d, proxyAddr)
	}
	sb.WriteString("	return \"DIRECT\";\n}\n")
	return sb.String()
}
