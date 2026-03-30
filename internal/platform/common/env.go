package common

import (
	"net/url"
	"strings"
)

type ProxyEnv struct {
	HTTPProxy  string
	HTTPSProxy string
	ALLProxy   string
	NOProxy    string
}

func BuildProxyEnv(proxyURL string, existingNoProxy string) ProxyEnv {
	return ProxyEnv{
		HTTPProxy:  proxyURL,
		HTTPSProxy: proxyURL,
		ALLProxy:   proxyURL,
		NOProxy:    MergeNoProxy(existingNoProxy, "localhost,127.0.0.1"),
	}
}

func MergeNoProxy(existing string, required string) string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(existing+","+required, ",") {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return strings.Join(out, ",")
}

func ValidateProxyURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme != "" && u.Host != ""
}
