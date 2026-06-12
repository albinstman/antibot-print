package antibot

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// regexForTest compiles the embedded presence artifact. The artifact is generated
// from the current signatures by cmd/gen (and gitignored), so tests run against the
// freshly built regex — not a stale committed copy.
func regexForTest(t *testing.T) *regexp.Regexp {
	t.Helper()
	return regexp.MustCompile(strings.TrimSpace(embeddedRegex))
}

// challengeRegexForTest compiles the embedded challenge-only artifact.
func challengeRegexForTest(t *testing.T) *regexp.Regexp {
	t.Helper()
	return regexp.MustCompile(strings.TrimSpace(embeddedChallengeRegex))
}

// blockRegexForTest compiles the embedded block-only artifact.
func blockRegexForTest(t *testing.T) *regexp.Regexp {
	t.Helper()
	return regexp.MustCompile(strings.TrimSpace(embeddedBlockRegex))
}

func TestDetect(t *testing.T) {
	re := regexForTest(t)
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{
			"cloudflare challenge",
			"HTTP/2 403 Forbidden\r\nServer: cloudflare\r\nCF-RAY: abc-LHR\r\n" +
				"Set-Cookie: __cf_bm=z; path=/\r\n\r\n<title>Attention Required! | Cloudflare</title>",
			[]string{"cloudflare"},
		},
		{
			"multi-vendor: cloudflare + hcaptcha",
			"HTTP/1.1 403\r\nSet-Cookie: __cf_bm=z; path=/\r\n\r\n" +
				`<div class="h-captcha"></div><script src="https://js.hcaptcha.com/1/api.js"></script>`,
			[]string{"cloudflare", "hcaptcha"},
		},
		{
			"akamai cookie + recaptcha embed",
			"HTTP/1.1 200 OK\r\nSet-Cookie: _abck=1; path=/\r\n\r\n" +
				`<script src="https://www.google.com/recaptcha/api.js"></script>`,
			[]string{"akamai", "recaptcha"},
		},
		{
			// Akamai's primary signal: the dynamic script endpoint embedded in
			// the page (no cookies set yet on the first GET).
			"akamai script endpoint in body",
			"HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n" +
				`<script type="text/javascript" src="/NW8v7h/PL/5Y/ju4o/cXo1d3-69Ymik/f07uLShic9/MzUWAQ/KzdU/EGoXBRV2" defer></script>`,
			[]string{"akamai"},
		},
		{
			"benign nginx (negative)",
			"HTTP/1.1 200 OK\r\nServer: nginx\r\n\r\n<html>hello world</html>",
			[]string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect([]byte(tc.raw), re)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestChallengeDetect checks the challenge tier reports a vendor only when the
// response actually carries a challenge marker, not on mere presence.
func TestChallengeDetect(t *testing.T) {
	re := challengeRegexForTest(t)
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{
			// DataDome cookie alone = presence, not a challenge.
			"datadome presence only (negative)",
			"HTTP/1.1 200 OK\r\nSet-Cookie: datadome=abc; path=/\r\n\r\n<html>content</html>",
			[]string{},
		},
		{
			// The 403 interstitial references captcha-delivery = challenge.
			"datadome challenge",
			"HTTP/1.1 403 Forbidden\r\n\r\n" +
				`<script src="https://ct.captcha-delivery.com/c.js"></script>`,
			[]string{"datadome"},
		},
		{
			// __cf_bm cookie alone = presence, not a challenge.
			"cloudflare presence only (negative)",
			"HTTP/1.1 200 OK\r\nSet-Cookie: __cf_bm=z; path=/\r\n\r\n<html>content</html>",
			[]string{},
		},
		{
			"cloudflare challenge",
			"HTTP/1.1 403\r\ncf-mitigated: challenge\r\n\r\n<html></html>",
			[]string{"cloudflare"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect([]byte(tc.raw), re)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestBlockDetect checks the block tier reports a vendor only when the response
// carries a hard-block marker — not on presence, and not on a solvable challenge.
func TestBlockDetect(t *testing.T) {
	re := blockRegexForTest(t)
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{
			// Akamai's deny reference ID (code 18) survives customer-branded,
			// localized deny pages — here the Swedish label seen on adidas.se.
			"akamai hard block (localized custom page)",
			"HTTP/2.0 403 Forbidden\r\nServer: AkamaiNetStorage\r\n\r\n" +
				"<html><span>Referensfel: 18.d09c9bd5.1781276226.e9310d3</span></html>",
			[]string{"akamai"},
		},
		{
			// _abck cookie alone = presence, not a block.
			"akamai presence only (negative)",
			"HTTP/1.1 200 OK\r\nSet-Cookie: _abck=1; path=/\r\n\r\n<html>content</html>",
			[]string{},
		},
		{
			// A solvable challenge interstitial = challenge tier, not a block.
			"akamai challenge only (negative)",
			"HTTP/1.1 200 OK\r\n\r\n<html><div id=\"sec-if-cpt-container\"></div></html>",
			[]string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect([]byte(tc.raw), re)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}
