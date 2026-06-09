// Direct-fetch path: retrieve a URL ourselves and feed the response to detection.
//
// Two fingerprints are available, because the request shape changes what antibots
// reveal in opposite directions:
//
//   - browser (default): a Chrome TLS/HTTP-2 fingerprint and header set, so
//     evasion-aware WAFs (Akamai, DataDome, …) reveal their cookies/scripts.
//   - naive (--naive): Go's stdlib client with its default Go TLS/HTTP fingerprint
//     and no browser headers, so challenge-on-suspicion vendors (e.g. PerimeterX)
//     that pass clean browsers silently still serve their block page to an obvious bot.
//
// Either way we follow redirects manually, capturing each hop's raw response and
// concatenating them (like `curl -i -L`) so intermediate Set-Cookie/headers — where
// challenges are frequently planted — survive into Normalize's multi-block parsing.
package antibot

import (
	"bytes"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// defaultProfile is the browser profile used when --profile is not given. Chrome's
// TLS/HTTP-2 fingerprint has been stable since Chrome 146, so chrome_147 and
// chrome_148 — which bogdanfinn/tls-client does not ship — are synthesized from
// chrome_146's fingerprint with only the User-Agent bumped (see resolveProfile).
const defaultProfile = "chrome_148"

// maxRedirects caps the manually-followed redirect chain.
const maxRedirects = 10

const fetchTimeout = 30 * time.Second

// hopResponse is the minimal view of one HTTP response the fetch loop needs,
// decoupled from whichever client (browser-impersonating or stdlib) produced it.
type hopResponse struct {
	proto      string
	statusCode int
	status     string
	header     map[string][]string
	body       []byte
}

// doer performs a single GET without following redirects.
type doer func(rawURL string) (*hopResponse, error)

// fetch retrieves rawURL, following redirects manually, and returns the whole chain
// as a raw `curl -i -L`-style byte stream ready for Detect/Normalize. When naive is
// true it uses Go's stdlib client (default Go fingerprint, no browser headers);
// otherwise it impersonates the given browser profile.
func fetch(rawURL, profileName string, naive bool) ([]byte, error) {
	var (
		do  doer
		err error
	)
	if naive {
		do, err = naiveDoer()
	} else {
		do, err = browserDoer(profileName)
	}
	if err != nil {
		return nil, err
	}
	return fetchChain(rawURL, do)
}

// fetchChain drives the redirect loop with do and returns the concatenated raw chain.
func fetchChain(rawURL string, do doer) ([]byte, error) {
	var chain bytes.Buffer
	current := rawURL
	for hop := 0; ; hop++ {
		resp, err := do(current)
		if err != nil {
			return nil, err
		}
		writeRawResponse(&chain, resp)

		loc := headerGet(resp.header, "Location")
		if !isRedirect(resp.statusCode) || loc == "" || hop >= maxRedirects {
			break
		}
		next, err := resolveLocation(current, loc)
		if err != nil {
			break // malformed Location: stop here, we already captured this hop
		}
		current = next
	}
	return chain.Bytes(), nil
}

// browserDoer builds a tls-client doer that impersonates the named browser profile,
// sending that browser's header set in its native order across a shared cookie jar.
func browserDoer(profileName string) (doer, error) {
	tlsProfile, header, err := resolveProfile(profileName)
	if err != nil {
		return nil, err
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithClientProfile(tlsProfile),
		tls_client.WithTimeoutSeconds(int(fetchTimeout/time.Second)),
		tls_client.WithCookieJar(tls_client.NewCookieJar()), // carry challenge cookies across hops
		tls_client.WithNotFollowRedirects(),                 // we drive the chain ourselves
		tls_client.WithRandomTLSExtensionOrder(),            // permute extension order per connection, like real Chrome
	)
	if err != nil {
		return nil, err
	}
	return func(rawURL string) (*hopResponse, error) {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header = header()
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
		}
		body, err := io.ReadAll(resp.Body) // auto-decompressed (gzip/deflate/br/zstd) by the transport
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading body of %s: %w", rawURL, err)
		}
		return &hopResponse{resp.Proto, resp.StatusCode, resp.Status, map[string][]string(resp.Header), body}, nil
	}, nil
}

// naiveDoer builds a stdlib net/http doer: Go's default TLS/HTTP fingerprint and
// default headers (Go-http-client User-Agent, auto gzip), redirects left to us.
func naiveDoer() (doer, error) {
	client := &stdhttp.Client{
		Timeout: fetchTimeout,
		CheckRedirect: func(*stdhttp.Request, []*stdhttp.Request) error {
			return stdhttp.ErrUseLastResponse
		},
	}
	return func(rawURL string) (*hopResponse, error) {
		resp, err := client.Get(rawURL)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
		}
		body, err := io.ReadAll(resp.Body) // auto-decompressed (gzip) by the transport
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading body of %s: %w", rawURL, err)
		}
		return &hopResponse{resp.Proto, resp.StatusCode, resp.Status, map[string][]string(resp.Header), body}, nil
	}, nil
}

// resolveProfile maps a --profile name to the tls-client ClientProfile that drives
// its TLS/HTTP-2 fingerprint and a builder for the header set to send with it.
//
// Chrome's ClientHello and HTTP/2 settings have not changed since Chrome 146, so
// chrome_147 and chrome_148 — which bogdanfinn/tls-client does not ship — reuse
// chrome_146's fingerprint and differ only in their User-Agent. Every other name is
// looked up in the library's profile table; its header set is chosen by browser family.
func resolveProfile(name string) (profiles.ClientProfile, func() http.Header, error) {
	switch name {
	case "chrome_147", "chrome_148":
		v := majorVersion(name)
		return profiles.Chrome_146, func() http.Header { return chromeHeader(v) }, nil
	}
	p, ok := profiles.MappedTLSClients[name]
	if !ok {
		return profiles.ClientProfile{}, nil, fmt.Errorf(
			"unknown profile %q (chrome_147/chrome_148, or any name from github.com/bogdanfinn/tls-client/profiles)", name)
	}
	return p, headerFor(name), nil
}

// headerFor returns the header builder matching a profile's browser family. Firefox
// gets its own set (no Chromium client hints); everything else (chrome, edge, opera,
// …) gets the Chromium navigation header set. The version is taken from the name.
func headerFor(name string) func() http.Header {
	v := majorVersion(name)
	if strings.HasPrefix(name, "firefox") {
		return func() http.Header { return firefoxHeader(v) }
	}
	if v == 0 {
		v = majorVersion(defaultProfile)
	}
	return func() http.Header { return chromeHeader(v) }
}

// majorVersion extracts the first run of digits from a profile name
// ("chrome_148" → 148, "firefox_135" → 135); 0 if the name carries no version.
func majorVersion(name string) int {
	i := strings.IndexFunc(name, func(r rune) bool { return r >= '0' && r <= '9' })
	if i < 0 {
		return 0
	}
	j := i
	for j < len(name) && name[j] >= '0' && name[j] <= '9' {
		j++
	}
	v, _ := strconv.Atoi(name[i:j])
	return v
}

// chromeHeader returns Chrome's navigation request headers in the order Chrome sends
// them, with the given major version stamped into the User-Agent and client hints.
// The header set is stable across recent Chrome majors; only the version moves. The
// HeaderOrderKey magic key tells fhttp to emit them in this order; the HTTP/2
// pseudo-header order and SETTINGS frame come from the client profile itself.
func chromeHeader(major int) http.Header {
	return http.Header{
		"sec-ch-ua":                 {fmt.Sprintf(`"Chromium";v="%d", "Google Chrome";v="%d", "Not/A)Brand";v="99"`, major, major)},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.0.0 Safari/537.36", major)},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"accept-language":           {"en-US,en;q=0.9"},
		"priority":                  {"u=0, i"},
		http.HeaderOrderKey: {
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"upgrade-insecure-requests", "user-agent", "accept",
			"sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest",
			"accept-encoding", "accept-language", "priority",
		},
	}
}

// firefoxHeader returns Firefox's navigation request headers in Firefox's order, with
// the given major version stamped into the User-Agent. Firefox sends no Chromium
// client hints (sec-ch-ua*); this is a best-effort set — refine against a real
// capture if exact Firefox parity is needed.
func firefoxHeader(major int) http.Header {
	return http.Header{
		"user-agent":                {fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:%d.0) Gecko/20100101 Firefox/%d.0", major, major)},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/png,image/svg+xml,*/*;q=0.8"},
		"accept-language":           {"en-US,en;q=0.5"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"upgrade-insecure-requests": {"1"},
		"sec-fetch-dest":            {"document"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-user":            {"?1"},
		"priority":                  {"u=0, i"},
		http.HeaderOrderKey: {
			"user-agent", "accept", "accept-language", "accept-encoding",
			"upgrade-insecure-requests", "sec-fetch-dest", "sec-fetch-mode",
			"sec-fetch-site", "sec-fetch-user", "priority",
		},
	}
}

// writeRawResponse serializes one hop as raw HTTP: status line, headers, blank line,
// body. This is exactly the shape Normalize parses from piped `curl -i` output.
func writeRawResponse(buf *bytes.Buffer, resp *hopResponse) {
	proto := resp.proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	status := resp.status
	if status == "" {
		status = fmt.Sprintf("%d %s", resp.statusCode, stdhttp.StatusText(resp.statusCode))
	}
	fmt.Fprintf(buf, "%s %s\r\n", proto, status)
	for key, vals := range resp.header {
		if key == "" || key[0] == ':' {
			continue // skip any HTTP/2 pseudo-headers
		}
		for _, v := range vals {
			fmt.Fprintf(buf, "%s: %s\r\n", key, v)
		}
	}
	buf.WriteString("\r\n")
	buf.Write(resp.body)
	buf.WriteString("\r\n")
}

// headerGet does a case-insensitive header lookup (HTTP/2 responses lowercase keys).
func headerGet(h map[string][]string, key string) string {
	if v := h[key]; len(v) > 0 {
		return v[0]
	}
	for k, v := range h {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func isRedirect(code int) bool {
	switch code {
	case 301, 302, 303, 307, 308:
		return true
	}
	return false
}

// resolveLocation resolves a (possibly relative) Location against the current URL.
func resolveLocation(base, loc string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	l, err := url.Parse(loc)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(l).String(), nil
}
