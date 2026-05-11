package gateway

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type networkInterfacesTool struct{}

func (t *networkInterfacesTool) ID() string             { return "net.interfaces" }
func (t *networkInterfacesTool) Domain() string         { return "network" }
func (t *networkInterfacesTool) Action() string         { return "inspect_interfaces" }
func (t *networkInterfacesTool) RiskClass() string      { return "read_only" }
func (t *networkInterfacesTool) ExecutionLevel() string { return "L0" }
func (t *networkInterfacesTool) Executes() bool         { return false }
func (t *networkInterfacesTool) UsesNetwork() bool      { return false }
func (t *networkInterfacesTool) WriteIntent() bool      { return false }
func (t *networkInterfacesTool) Description() string    { return "Inspect local network interfaces" }
func (t *networkInterfacesTool) Execute(ctx context.Context, req Request) (Result, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return Result{}, err
	}
	out := make([]map[string]any, 0, len(ifaces))
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		a := []string{}
		for _, addr := range addrs {
			a = append(a, addr.String())
		}
		out = append(out, map[string]any{"name": iface.Name, "mtu": iface.MTU, "flags": iface.Flags.String(), "addrs": a})
	}
	return Result{Data: map[string]any{"interfaces": out, "count": len(out)}, Message: "interfaces listed"}, nil
}

type networkConnectivityTool struct{}

const maxConnectivityTargetBytes = 512

var errConnectivityTargetTooLarge = errors.New("net.connectivity target too large")

func normalizeConnectivityTarget(raw string) (string, error) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return "1.1.1.1:53", nil
	}
	if len(target) > maxConnectivityTargetBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errConnectivityTargetTooLarge, len(target), maxConnectivityTargetBytes)
	}
	return target, nil
}

func (t *networkConnectivityTool) ID() string             { return "net.connectivity" }
func (t *networkConnectivityTool) Domain() string         { return "network" }
func (t *networkConnectivityTool) Action() string         { return "test_connectivity" }
func (t *networkConnectivityTool) RiskClass() string      { return "scoped_execute" }
func (t *networkConnectivityTool) ExecutionLevel() string { return "L2" }
func (t *networkConnectivityTool) Executes() bool         { return false }
func (t *networkConnectivityTool) UsesNetwork() bool      { return true }
func (t *networkConnectivityTool) WriteIntent() bool      { return false }
func (t *networkConnectivityTool) Description() string    { return "Check TCP connectivity to host:port" }
func (t *networkConnectivityTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := normalizeConnectivityTarget(inputString(req.Input, "target"))
	if err != nil {
		return Result{}, err
	}
	timeoutMs := int(readFloat(req.Input, "timeoutMs", 5000))
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	conn, err := net.DialTimeout("tcp", target, time.Duration(timeoutMs)*time.Millisecond)
	ok := err == nil
	if conn != nil {
		_ = conn.Close()
	}
	return Result{Data: map[string]any{"target": target, "ok": ok, "error": errString(err)}, Message: "connectivity test complete"}, nil
}

type networkDNSLookupTool struct{}

const maxDNSLookupHostBytes = 253

var errDNSLookupHostTooLarge = errors.New("net.dns_lookup host too large")

func normalizeDNSLookupHost(raw string) (string, error) {
	host := strings.TrimSpace(raw)
	if host == "" {
		return "", errors.New("net.dns_lookup requires input.host")
	}
	if len(host) > maxDNSLookupHostBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errDNSLookupHostTooLarge, len(host), maxDNSLookupHostBytes)
	}
	return host, nil
}

func (t *networkDNSLookupTool) ID() string             { return "net.dns_lookup" }
func (t *networkDNSLookupTool) Domain() string         { return "network" }
func (t *networkDNSLookupTool) Action() string         { return "dns_lookup" }
func (t *networkDNSLookupTool) RiskClass() string      { return "read_only" }
func (t *networkDNSLookupTool) ExecutionLevel() string { return "L0" }
func (t *networkDNSLookupTool) Executes() bool         { return false }
func (t *networkDNSLookupTool) UsesNetwork() bool      { return true }
func (t *networkDNSLookupTool) WriteIntent() bool      { return false }
func (t *networkDNSLookupTool) Description() string    { return "Resolve DNS name to IP addresses" }
func (t *networkDNSLookupTool) Execute(ctx context.Context, req Request) (Result, error) {
	host, err := normalizeDNSLookupHost(inputString(req.Input, "host"))
	if err != nil {
		return Result{}, err
	}
	addrs, err := net.LookupHost(host)
	return Result{Data: map[string]any{"host": host, "addresses": addrs, "ok": err == nil, "error": errString(err)}, Message: "dns lookup complete"}, nil
}

type networkFetchTool struct{}

func (t *networkFetchTool) ID() string             { return "net.fetch" }
func (t *networkFetchTool) Domain() string         { return "network" }
func (t *networkFetchTool) Action() string         { return "fetch_url" }
func (t *networkFetchTool) RiskClass() string      { return "scoped_execute" }
func (t *networkFetchTool) ExecutionLevel() string { return "L2" }
func (t *networkFetchTool) Executes() bool         { return false }
func (t *networkFetchTool) UsesNetwork() bool      { return true }
func (t *networkFetchTool) WriteIntent() bool      { return false }
func (t *networkFetchTool) Description() string    { return "Fetch approved URL content (GET)" }
func (t *networkFetchTool) Execute(ctx context.Context, req Request) (Result, error) {
	raw, _ := req.Input["url"].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Result{}, errors.New("net.fetch requires input.url")
	}
	parsed, err := validateOutboundHTTPURL(ctx, raw, nil)
	if err != nil {
		return Result{}, err
	}
	client := newGuardedOutboundHTTPClient(ctx, nil)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Result{}, err
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, err := readGatewayHTTPResponseBody(resp.Body, "net.fetch", gatewayNetFetchResponseBodyLimit)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Data: map[string]any{
			"url":        parsed.String(),
			"statusCode": resp.StatusCode,
			"ok":         resp.StatusCode >= 200 && resp.StatusCode < 300,
			"body":       string(body),
		},
		Message: "url fetched",
	}, nil
}

type outboundHTTPResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

const maxOutboundHTTPURLBytes = 8 << 10

var errOutboundHTTPURLTooLarge = errors.New("outbound HTTP URL too large")

func validateOutboundHTTPURL(ctx context.Context, raw string, resolver outboundHTTPResolver) (*url.URL, error) {
	normalized := strings.TrimSpace(raw)
	if len(normalized) > maxOutboundHTTPURLBytes {
		return nil, fmt.Errorf("%w: %d > %d bytes", errOutboundHTTPURLTooLarge, len(normalized), maxOutboundHTTPURLBytes)
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("only http/https URLs are allowed")
	}
	if parsed.User != nil {
		return nil, errors.New("URL userinfo is not allowed")
	}
	host := parsed.Hostname()
	if strings.TrimSpace(host) == "" {
		return nil, errors.New("URL host is required")
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if _, err := resolveAllowedOutboundIPs(ctx, host, resolver); err != nil {
		return nil, err
	}
	return parsed, nil
}

func resolveAllowedOutboundIPs(ctx context.Context, host string, resolver outboundHTTPResolver) ([]net.IP, error) {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if ip := net.ParseIP(host); ip != nil {
		if blockedOutboundIP(ip) {
			return nil, fmt.Errorf("blocked outbound target: %s", host)
		}
		return []net.IP{ip}, nil
	}
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve outbound target %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("resolve outbound target %q: no addresses", host)
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		if addr.IP == nil || blockedOutboundIP(addr.IP) {
			return nil, fmt.Errorf("blocked outbound target: %s", host)
		}
		ips = append(ips, addr.IP)
	}
	return ips, nil
}

func validateOutboundHTTPRedirect(ctx context.Context, req *http.Request, resolver outboundHTTPResolver) error {
	if req == nil || req.URL == nil {
		return errors.New("redirect URL is required")
	}
	_, err := validateOutboundHTTPURL(ctx, req.URL.String(), resolver)
	return err
}

func newGuardedOutboundHTTPClient(ctx context.Context, resolver outboundHTTPResolver) *http.Client {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(dialCtx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := resolveAllowedOutboundIPs(dialCtx, host, resolver)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, ip := range ips {
			conn, err := dialer.DialContext(dialCtx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("resolve outbound target %q: no addresses", host)
	}
	return &http.Client{
		Timeout:   20 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return validateOutboundHTTPRedirect(ctx, req, resolver)
		},
	}
}

func blockedOutboundIP(ip net.IP) bool {
	return ip == nil ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast()
}

type webSearchTool struct{}

const maxWebSearchQueryBytes = 2 << 10

var errWebSearchQueryTooLarge = errors.New("web.search query too large")

func normalizeWebSearchQuery(raw string) (string, error) {
	query := strings.TrimSpace(raw)
	if query == "" {
		return "", errors.New("web.search requires input.query")
	}
	if len(query) > maxWebSearchQueryBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errWebSearchQueryTooLarge, len(query), maxWebSearchQueryBytes)
	}
	return query, nil
}

func (t *webSearchTool) ID() string             { return "web.search" }
func (t *webSearchTool) Domain() string         { return "network" }
func (t *webSearchTool) Action() string         { return "search_web" }
func (t *webSearchTool) RiskClass() string      { return "read_only" }
func (t *webSearchTool) ExecutionLevel() string { return "L2" }
func (t *webSearchTool) Executes() bool         { return false }
func (t *webSearchTool) UsesNetwork() bool      { return true }
func (t *webSearchTool) WriteIntent() bool      { return false }
func (t *webSearchTool) Description() string {
	return "Search the public web and return compact result titles, URLs, and snippets"
}
func (t *webSearchTool) Execute(ctx context.Context, req Request) (Result, error) {
	query, err := normalizeWebSearchQuery(inputString(req.Input, "query"))
	if err != nil {
		return Result{}, err
	}
	limit := 5
	if v, ok := req.Input["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 10 {
		limit = 10
	}
	searchURL := "https://duckduckgo.com/html/?q=" + url.QueryEscape(query)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return Result{}, err
	}
	httpReq.Header.Set("User-Agent", "FORGE/1.0 web.search")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(httpReq)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, err := readGatewayHTTPResponseBody(resp.Body, "web.search", gatewayWebSearchResponseBodyLimit)
	if err != nil {
		return Result{}, err
	}
	results := parseDuckDuckGoHTMLResults(string(body), limit)
	return Result{
		Data: map[string]any{
			"query":      query,
			"searchUrl":  searchURL,
			"statusCode": resp.StatusCode,
			"ok":         resp.StatusCode >= 200 && resp.StatusCode < 300,
			"results":    results,
			"count":      len(results),
		},
		Message: "web search completed",
	}, nil
}

func parseDuckDuckGoHTMLResults(rawHTML string, limit int) []map[string]any {
	out := make([]map[string]any, 0, limit)
	blocks := strings.Split(rawHTML, "result__body")
	for _, block := range blocks {
		if len(out) >= limit {
			break
		}
		title := htmlText(firstRegexGroup(block, `(?s)class="result__a"[^>]*>(.*?)</a>`))
		href := htmlEntityDecode(firstRegexGroup(block, `(?s)class="result__a"[^>]*href="([^"]+)"`))
		snippet := htmlText(firstRegexGroup(block, `(?s)class="result__snippet"[^>]*>(.*?)</a>`))
		if snippet == "" {
			snippet = htmlText(firstRegexGroup(block, `(?s)class="result__snippet"[^>]*>(.*?)</div>`))
		}
		href = normalizeDuckDuckGoResultURL(href)
		if title == "" || href == "" {
			continue
		}
		out = append(out, map[string]any{"title": title, "url": href, "snippet": snippet})
	}
	return out
}

func firstRegexGroup(s, pattern string) string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func htmlText(s string) string {
	if s == "" {
		return ""
	}
	tagRE := regexp.MustCompile(`(?s)<[^>]+>`)
	text := tagRE.ReplaceAllString(s, " ")
	text = htmlEntityDecode(text)
	return strings.Join(strings.Fields(text), " ")
}

func htmlEntityDecode(s string) string {
	replacer := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&#x27;", "'")
	return replacer.Replace(s)
}

func normalizeDuckDuckGoResultURL(raw string) string {
	raw = strings.TrimSpace(htmlEntityDecode(raw))
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Host != "" && strings.Contains(parsed.Host, "duckduckgo.com") {
		if uddg := parsed.Query().Get("uddg"); uddg != "" {
			if decoded, err := url.QueryUnescape(uddg); err == nil {
				return decoded
			}
			return uddg
		}
	}
	return raw
}
