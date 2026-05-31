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
package main

import (
	"bytes"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// defaultProfile is the tls-client profile used when --profile is not given. Bump
// this to the newest Chrome as bogdanfinn/tls-client adds profiles; the matching
// header set below should be updated in lockstep.
const defaultProfile = "chrome_146"

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
// sending Chrome's header set in Chrome's order across a shared cookie jar.
func browserDoer(profileName string) (doer, error) {
	profile, ok := profiles.MappedTLSClients[profileName]
	if !ok {
		return nil, fmt.Errorf("unknown profile %q (see github.com/bogdanfinn/tls-client/profiles for names)", profileName)
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithClientProfile(profile),
		tls_client.WithTimeoutSeconds(int(fetchTimeout/time.Second)),
		tls_client.WithCookieJar(tls_client.NewCookieJar()), // carry challenge cookies across hops
		tls_client.WithNotFollowRedirects(),                 // we drive the chain ourselves
	)
	if err != nil {
		return nil, err
	}
	return func(rawURL string) (*hopResponse, error) {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header = chromeRequestHeader()
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

// chromeRequestHeader returns Chrome's navigation request headers in the order Chrome
// sends them. The HeaderOrderKey magic key tells fhttp to emit them in this order; the
// HTTP/2 pseudo-header order and SETTINGS frame come from the client profile itself.
func chromeRequestHeader() http.Header {
	return http.Header{
		"sec-ch-ua":                 {`"Chromium";v="146", "Google Chrome";v="146", "Not?A_Brand";v="99"`},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"},
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
