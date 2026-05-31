package main

import (
	"reflect"
	"regexp"
	"testing"
)

// regexForTest compiles fresh from the signature sources so tests check the
// current signatures, not a possibly-stale embedded artifact.
func regexForTest(t *testing.T) *regexp.Regexp {
	t.Helper()
	pattern, err := CompileSignatures("signatures", "")
	if err != nil {
		t.Fatalf("compile signatures: %v", err)
	}
	return regexp.MustCompile(pattern)
}

// challengeRegexForTest compiles the challenge-only tier fresh from sources.
func challengeRegexForTest(t *testing.T) *regexp.Regexp {
	t.Helper()
	pattern, err := CompileChallengeSignatures("signatures", "")
	if err != nil {
		t.Fatalf("compile challenge signatures: %v", err)
	}
	return regexp.MustCompile(pattern)
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

// TestArtifactInSync guards against committing signatures without regenerating
// the embedded regex artifacts.
func TestArtifactInSync(t *testing.T) {
	pattern, err := CompileSignatures("signatures", "")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	embedded := regexp.MustCompile(`\s+$`).ReplaceAllString(embeddedRegex, "")
	if embedded != pattern {
		t.Error("antibot.re2.txt is stale — run: go run . compile")
	}

	chPattern, err := CompileChallengeSignatures("signatures", "")
	if err != nil {
		t.Fatalf("compile challenge: %v", err)
	}
	embeddedCh := regexp.MustCompile(`\s+$`).ReplaceAllString(embeddedChallengeRegex, "")
	if embeddedCh != chPattern {
		t.Error("antibot-challenge.re2.txt is stale — run: go run . compile")
	}
}
