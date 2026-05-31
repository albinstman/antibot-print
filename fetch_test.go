package main

import (
	"bytes"
	"reflect"
	"regexp"
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

func TestIsRedirect(t *testing.T) {
	for _, c := range []struct {
		code int
		want bool
	}{{200, false}, {301, true}, {302, true}, {303, true}, {307, true}, {308, true}, {403, false}, {404, false}} {
		if got := isRedirect(c.code); got != c.want {
			t.Errorf("isRedirect(%d) = %v, want %v", c.code, got, c.want)
		}
	}
}

func TestHeaderGet(t *testing.T) {
	// HTTP/2 responses lowercase header keys; lookup must be case-insensitive.
	h := map[string][]string{"location": {"https://example.com/next"}}
	if got := headerGet(h, "Location"); got != "https://example.com/next" {
		t.Errorf("headerGet(lowercase location) = %q, want the URL", got)
	}
	if got := headerGet(h, "X-Absent"); got != "" {
		t.Errorf("headerGet(absent) = %q, want empty", got)
	}
}

func TestResolveLocation(t *testing.T) {
	for _, c := range []struct {
		base, loc, want string
	}{
		{"https://a.com/x", "https://b.com/y", "https://b.com/y"}, // absolute
		{"https://a.com/x/y", "/z", "https://a.com/z"},            // root-relative
		{"https://a.com/x/y", "z", "https://a.com/x/z"},           // path-relative
		{"https://a.com/x", "//cdn.com/p", "https://cdn.com/p"},   // scheme-relative
	} {
		got, err := resolveLocation(c.base, c.loc)
		if err != nil {
			t.Fatalf("resolveLocation(%q,%q): %v", c.base, c.loc, err)
		}
		if got != c.want {
			t.Errorf("resolveLocation(%q,%q) = %q, want %q", c.base, c.loc, got, c.want)
		}
	}
}

// TestChromeHeaderOrderComplete guards that every real header we send is listed in
// the HeaderOrderKey, so fhttp emits a complete, browser-ordered header block.
func TestChromeHeaderOrderComplete(t *testing.T) {
	h := chromeRequestHeader()
	order := map[string]bool{}
	for _, k := range h[http.HeaderOrderKey] {
		order[k] = true
	}
	for k := range h {
		if k == http.HeaderOrderKey {
			continue
		}
		if !order[k] {
			t.Errorf("header %q is sent but missing from HeaderOrderKey", k)
		}
	}
}

// TestWriteRawResponseRoundTrip verifies the raw response we assemble per hop is the
// shape Normalize/Detect expects — including a redirect-hop Set-Cookie that must
// survive into the captured chain.
func TestWriteRawResponseRoundTrip(t *testing.T) {
	pattern, err := CompileSignatures("signatures", "")
	if err != nil {
		t.Fatalf("compile signatures: %v", err)
	}
	re := regexp.MustCompile(pattern)

	var buf bytes.Buffer
	// hop 1: a 302 that plants the Akamai _abck cookie, then redirects.
	writeRawResponse(&buf, &hopResponse{
		proto:      "HTTP/2.0",
		statusCode: 302,
		status:     "302 Found",
		header: map[string][]string{
			"Location":   {"https://example.com/home"},
			"Set-Cookie": {"_abck=1; path=/"},
		},
	})
	// hop 2: the final 200 with a benign body (lowercase header key, as HTTP/2 sends).
	writeRawResponse(&buf, &hopResponse{
		proto:      "HTTP/2.0",
		statusCode: 200,
		status:     "200 OK",
		header:     map[string][]string{"content-type": {"text/html"}},
		body:       []byte("<html>home</html>"),
	})

	got := Detect(buf.Bytes(), re)
	if want := []string{"akamai"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Detect(chain) = %v, want %v (redirect-hop Set-Cookie should survive)", got, want)
	}
}
