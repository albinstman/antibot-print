package main

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
// detection tier, the normalized view, and the full raw response.
func TestWriteDebugFull(t *testing.T) {
	raw := "HTTP/1.1 403\r\ncf-mitigated: challenge\r\n\r\n<html>blocked</html>"
	var buf bytes.Buffer
	writeDebug(&buf, []byte(raw), debugContext{url: "https://example.com", profile: "chrome_146"}, true, false)
	out := buf.String()

	for _, want := range []string{
		"request:",
		"url:    https://example.com",
		"mode:   browser (profile chrome_146)",
		"detection (presence):",
		"cloudflare",
		"normalized (what the regex matches against):",
		"S:403",
		"raw response:",
		"<html>blocked</html>", // full raw response is included verbatim
	} {
		if !strings.Contains(out, want) {
			t.Errorf("full debug output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

// TestWriteDebugLight checks the light report keeps the small sections but omits
// the two bulky ones (normalized view + raw response).
func TestWriteDebugLight(t *testing.T) {
	raw := "HTTP/1.1 403\r\ncf-mitigated: challenge\r\n\r\n<html>blocked</html>"
	var buf bytes.Buffer
	writeDebug(&buf, []byte(raw), debugContext{url: "https://example.com", profile: "chrome_146"}, false, false)
	out := buf.String()

	for _, want := range []string{"detection (presence):", "cloudflare"} {
		if !strings.Contains(out, want) {
			t.Errorf("light debug output missing %q\n--- got ---\n%s", want, out)
		}
	}
	for _, absent := range []string{"normalized (what the regex matches against):", "raw response:", "<html>blocked</html>"} {
		if strings.Contains(out, absent) {
			t.Errorf("light debug output should omit %q\n--- got ---\n%s", absent, out)
		}
	}
}

// TestWriteDebugTier checks the report shows only the tier the run uses: the
// challenge tier under -c, the presence tier otherwise.
func TestWriteDebugTier(t *testing.T) {
	raw := "HTTP/1.1 403\r\ncf-mitigated: challenge\r\n\r\n<html>blocked</html>"
	var presence, challenge bytes.Buffer
	writeDebug(&presence, []byte(raw), debugContext{fromStdin: true}, false, false)
	writeDebug(&challenge, []byte(raw), debugContext{fromStdin: true}, false, true)

	if p := presence.String(); !strings.Contains(p, "detection (presence):") || strings.Contains(p, "detection (challenge):") {
		t.Errorf("default run should report only the presence tier, got:\n%s", p)
	}
	if c := challenge.String(); !strings.Contains(c, "detection (challenge):") || strings.Contains(c, "detection (presence):") {
		t.Errorf("-c run should report only the challenge tier, got:\n%s", c)
	}
}

// TestWriteDebugStdin checks the stdin source is labeled and no fetch mode leaks in.
func TestWriteDebugStdin(t *testing.T) {
	var buf bytes.Buffer
	writeDebug(&buf, []byte("HTTP/1.1 200 OK\r\nServer: nginx\r\n\r\nhi"), debugContext{fromStdin: true}, true, false)
	out := buf.String()
	if !strings.Contains(out, "source: stdin") {
		t.Errorf("expected stdin source label, got:\n%s", out)
	}
	if strings.Contains(out, "mode:") {
		t.Errorf("stdin debug should not report a fetch mode, got:\n%s", out)
	}
}
