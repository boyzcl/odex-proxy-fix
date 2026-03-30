package detect

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ProxyCandidate struct {
	URL       string   `json:"url"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Source    string   `json:"source"`
	Listening bool     `json:"listening"`
	Verified  bool     `json:"verified"`
	Score     int      `json:"score"`
	Errors    []string `json:"errors,omitempty"`
}

type ProxyOptions struct {
	ExplicitPort int
	CheckURLs    []string
}

func DetectProxyCandidates(opts ProxyOptions) []ProxyCandidate {
	type key struct {
		host string
		port int
	}
	seen := map[key]*ProxyCandidate{}
	add := func(host string, port int, source string, baseScore int) {
		if port <= 0 {
			return
		}
		k := key{host: host, port: port}
		if existing, ok := seen[k]; ok {
			existing.Score += baseScore
			if existing.Source == "" {
				existing.Source = source
			}
			return
		}
		seen[k] = &ProxyCandidate{
			URL:    fmt.Sprintf("http://%s:%d", host, port),
			Host:   host,
			Port:   port,
			Source: source,
			Score:  baseScore,
		}
	}

	if opts.ExplicitPort > 0 {
		add("127.0.0.1", opts.ExplicitPort, "flag", 100)
	}
	for _, envKey := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		if raw := os.Getenv(envKey); raw != "" {
			if host, port, ok := parseProxyURL(raw); ok {
				add(host, port, "env:"+envKey, 40)
			}
		}
	}
	for _, port := range []int{7897, 7890, 1087, 1080, 8080, 3128} {
		add("127.0.0.1", port, "common-port", 0)
	}

	checkURLs := opts.CheckURLs
	if len(checkURLs) == 0 {
		checkURLs = []string{
			"https://docs.openai.com/",
			"https://www.google.com/generate_204",
			"https://1.1.1.1/",
		}
	}

	var out []ProxyCandidate
	for _, c := range seen {
		c.Listening = canDial(c.Host, c.Port)
		if !c.Listening {
			c.Errors = append(c.Errors, "port not listening")
			c.Score -= 30
			out = append(out, *c)
			continue
		}
		c.Verified, c.Errors = verifyProxy(c.URL, checkURLs)
		if c.Verified {
			c.Score += 50
		} else {
			c.Score -= 30
		}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Port < out[j].Port
		}
		return out[i].Score > out[j].Score
	})
	return out
}

func BestProxy(candidates []ProxyCandidate) (ProxyCandidate, bool) {
	for _, c := range candidates {
		if c.Verified {
			return c, true
		}
	}
	return ProxyCandidate{}, false
}

func canDial(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func verifyProxy(proxyURL string, checkURLs []string) (bool, []string) {
	var errs []string
	pu, err := url.Parse(proxyURL)
	if err != nil {
		return false, []string{err.Error()}
	}
	for _, target := range checkURLs {
		ok, err := headViaProxy(pu, target)
		if ok {
			return true, nil
		}
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return false, errs
}

func headViaProxy(proxyURL *url.URL, target string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, target, nil)
	if err != nil {
		return false, err
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	resp, err := client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		return true, nil
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false, err
	}
	resp, err = client.Do(req)
	if err != nil {
		return false, err
	}
	_ = resp.Body.Close()
	return true, nil
}

func parseProxyURL(raw string) (string, int, bool) {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", 0, false
	}
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil || host == "" {
		return "", 0, false
	}
	return host, port, true
}
