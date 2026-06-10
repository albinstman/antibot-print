package antibot

import (
	"bytes"
	"strings"
	"testing"
)

// TestDetectVerbose checks the verbose matcher reports the vendor and keeps the
// exact substring that triggered it (the "what it detects" view).
func TestDetectVerbose(t *testing.T) {
	re := regexForTest(t)
	raw := "HTTP/2 403 Forbidden\r\nServer: cloudflare\r\nSet-Cookie: __cf_bm=z; path=/\r\n\r\n" +
		`<div class="h-captcha"></div>`
	norm := Normalize([]byte(raw), DefaultBodyCap)

	got := detectVerbose(norm, re)
	if len(got) != 2 || got[0].vendor != "cloudflare" || got[1].vendor != "hcaptcha" {
		t.Fatalf("detectVerbose vendors = %+v, want cloudflare,hcaptcha", got)
	}
	// Every reported match must be a real substring of what the regex scanned.
	for _, vm := range got {
		if len(vm.matched) == 0 {
			t.Errorf("%s: no matched text recorded", vm.vendor)
		}
		for _, m := range vm.matched {
			if !strings.Contains(norm, m) {
				t.Errorf("%s: matched %q is not a substring of the normalized input", vm.vendor, m)
			}
		}
	}
}

// TestWriteDebugFull checks the full diagnostic covers the request, the active
// detection tier, and the full raw response.
func TestWriteDebugFull(t *testing.T) {
	raw := "HTTP/1.1 403\r\ncf-mitigated: challenge\r\n\r\n<html>blocked</html>"
	var buf bytes.Buffer
	writeDebug(&buf, []byte(raw), debugContext{url: "https://example.com", profile: "chrome_146"}, true)
	out := buf.String()

	for _, want := range []string{
		"request:",
		"url:    https://example.com",
		"mode:   browser (profile chrome_146)",
		"detection (presence):",
		"cloudflare",
		"raw response:",
		"<html>blocked</html>", // full raw response is included verbatim
	} {
		if !strings.Contains(out, want) {
			t.Errorf("full debug output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

// TestWriteDebugLight checks the light report keeps the small sections but omits
// the raw response.
func TestWriteDebugLight(t *testing.T) {
	raw := "HTTP/1.1 403\r\ncf-mitigated: challenge\r\n\r\n<html>blocked</html>"
	var buf bytes.Buffer
	writeDebug(&buf, []byte(raw), debugContext{url: "https://example.com", profile: "chrome_146"}, false)
	out := buf.String()

	for _, want := range []string{"detection (presence):", "cloudflare"} {
		if !strings.Contains(out, want) {
			t.Errorf("light debug output missing %q\n--- got ---\n%s", want, out)
		}
	}
	for _, absent := range []string{"raw response:", "<html>blocked</html>"} {
		if strings.Contains(out, absent) {
			t.Errorf("light debug output should omit %q\n--- got ---\n%s", absent, out)
		}
	}
}

// TestWriteDebugTier checks the report always shows both tiers — presence and
// challenge — regardless of -c.
func TestWriteDebugTier(t *testing.T) {
	raw := "HTTP/1.1 403\r\ncf-mitigated: challenge\r\n\r\n<html>blocked</html>"
	var buf bytes.Buffer
	writeDebug(&buf, []byte(raw), debugContext{fromStdin: true}, false)

	out := buf.String()
	for _, want := range []string{"detection (presence):", "detection (challenge):"} {
		if !strings.Contains(out, want) {
			t.Errorf("debug output missing %q, got:\n%s", want, out)
		}
	}
}

// TestParseHops checks the chain parser pulls each hop's status and Location.
func TestParseHops(t *testing.T) {
	raw := "HTTP/1.1 301 Moved Permanently\r\nLocation: https://www.example.com/\r\n\r\n" +
		"HTTP/1.1 200 OK\r\nServer: nginx\r\n\r\n<html>ok</html>"
	hops := parseHops([]byte(raw))
	if len(hops) != 2 {
		t.Fatalf("parseHops returned %d hops, want 2: %+v", len(hops), hops)
	}
	if hops[0].status != 301 || hops[0].location != "https://www.example.com/" {
		t.Errorf("hop 0 = %+v, want {301 https://www.example.com/}", hops[0])
	}
	if hops[1].status != 200 || hops[1].location != "" {
		t.Errorf("hop 1 = %+v, want {200 \"\"}", hops[1])
	}
}

// TestWriteDebugRedirectChain checks a multi-hop fetch reconstructs the visited
// URLs (resolving Location from the start URL), and a single hop omits the block.
func TestWriteDebugRedirectChain(t *testing.T) {
	raw := "HTTP/1.1 301 Moved Permanently\r\nLocation: https://www.example.com/\r\n\r\n" +
		"HTTP/1.1 200 OK\r\nServer: nginx\r\n\r\n<html>ok</html>"
	var buf bytes.Buffer
	writeDebug(&buf, []byte(raw), debugContext{url: "https://example.com", profile: "chrome_146"}, false)
	out := buf.String()
	if want := "redirects:\n    https://example.com (301) -> https://www.example.com/ (200)"; !strings.Contains(out, want) {
		t.Errorf("redirect chain missing %q\n--- got ---\n%s", want, out)
	}

	// A single-hop response has nothing to chain, so the block is omitted.
	var single bytes.Buffer
	writeDebug(&single, []byte("HTTP/1.1 200 OK\r\nServer: nginx\r\n\r\nhi"), debugContext{url: "https://example.com", profile: "chrome_146"}, false)
	if strings.Contains(single.String(), "redirects:") {
		t.Errorf("single-hop response should omit the redirects block, got:\n%s", single.String())
	}
}

// TestWriteDebugStdin checks the stdin source is labeled and no fetch mode leaks in.
func TestWriteDebugStdin(t *testing.T) {
	var buf bytes.Buffer
	writeDebug(&buf, []byte("HTTP/1.1 200 OK\r\nServer: nginx\r\n\r\nhi"), debugContext{fromStdin: true}, true)
	out := buf.String()
	if !strings.Contains(out, "source: stdin") {
		t.Errorf("expected stdin source label, got:\n%s", out)
	}
	if strings.Contains(out, "mode:") {
		t.Errorf("stdin debug should not report a fetch mode, got:\n%s", out)
	}
}
