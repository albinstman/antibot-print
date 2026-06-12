// Package antibot is the CLI: it names the antibot/WAF/CAPTCHA vendor(s) protecting
// a site from an HTTP response (fetched directly with a browser fingerprint, or read
// on stdin) by matching against the embedded RE2 artifacts. The regex artifacts are
// generated from signatures/ by package gen (see cmd/gen) and embedded by regex.go.
package antibot

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// version is the build version, supplied by the entrypoint (cmd/cli) via Run and
// stamped at release time with -ldflags "-X main.version=...". Defaults to "dev".
var version = "dev"

const usage = `antibot — name the antibot/WAF/CAPTCHA vendor(s) protecting a site.

Usage:
  antibot https://example.com        fetch the URL directly (browser fingerprint)
  curl -isS https://example.com | antibot   detect from a piped HTTP response

Prints one detected vendor slug per line (sorted). Exit status 0 if any vendor was
detected, 1 if none. When fetching directly, antibot sends a browser-like
TLS/HTTP-2 fingerprint and header set and follows redirects, so evasion-aware WAFs
reveal themselves. When piping, use 'curl -isS -L --compressed' to follow
redirects/decompress.

Options:
  -c, --challenge   only report vendors actively serving a challenge,
                    not mere vendor presence
  -b, --block       only report vendors serving a hard block (denied outright,
                    nothing to solve), not mere vendor presence
  -p, --profile P   browser profile to impersonate when fetching a URL (default %s;
                    e.g. chrome_146, firefox_135). chrome_147 and chrome_148 are
                    synthesized from chrome_146's fingerprint with a bumped User-Agent
  -n, --naive       fetch with Go's default (non-browser) TLS/HTTP fingerprint
                    instead of impersonating a browser — surfaces vendors that
                    challenge suspicious clients but pass real browsers silently
  -d, --debug       print a light diagnostic instead of the slug list: how the
                    response was fetched, the status chain, and every vendor
                    matched — in all tiers (presence, challenge and block),
                    regardless of -c/-b — with the exact text that triggered it
  -D, --debug-full  like --debug, plus the full raw response;
                    best redirected to a file (antibot -D URL > debug.txt)
  -r, --raw         print only the raw fetched response (status line, headers,
                    body — like 'curl -i -L'), no detection output; the exit code
                    still reflects detection (0 vendor found, 1 none)
  -o, --open        open the fetched HTML (final response body) in your default
                    browser, in addition to the normal output
  -h, --help        show this help and exit
  -V, --version     show version and exit

Commands:
  update            download and install the latest release (verifies checksum)

Environment:
  ANTIBOT_NO_UPDATE_CHECK   disable the daily "update available" check
`

// Run is the CLI entrypoint: it parses args, runs detection (or the update command),
// and returns the process exit code. ver is the build version from the binary's
// main package (empty keeps the default "dev").
func Run(ver string, args []string) int {
	if ver != "" {
		version = ver
	}
	if len(args) > 0 && args[0] == "update" {
		return runUpdate()
	}

	challenge := false
	block := false
	naive := false
	debug := false
	debugFull := false
	rawOnly := false
	open := false
	profile := defaultProfile
	profileSet := false
	url := ""
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-h" || a == "--help":
			fmt.Printf(usage, defaultProfile)
			return 0
		case a == "-V" || a == "--version":
			fmt.Printf("antibot %s\n", version)
			return 0
		case a == "-c" || a == "--challenge":
			challenge = true
		case a == "-b" || a == "--block":
			block = true
		case a == "-n" || a == "--naive":
			naive = true
		case a == "-d" || a == "--debug":
			debug = true
		case a == "-D" || a == "--debug-full":
			debugFull = true
		case a == "-r" || a == "--raw":
			rawOnly = true
		case a == "-o" || a == "--open":
			open = true
		case a == "-p" || a == "--profile":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "antibot: --profile requires a value")
				return 2
			}
			profile = args[i]
			profileSet = true
		case strings.HasPrefix(a, "-") && a != "-":
			fmt.Fprintf(os.Stderr, "antibot: unknown option %q (try --help)\n", a)
			return 2
		default:
			if url != "" {
				fmt.Fprintf(os.Stderr, "antibot: unexpected extra argument %q\n", a)
				return 2
			}
			url = a
		}
	}
	if naive && profileSet {
		fmt.Fprintln(os.Stderr, "antibot: --naive and --profile are mutually exclusive")
		return 2
	}
	if challenge && block {
		fmt.Fprintln(os.Stderr, "antibot: --challenge and --block are mutually exclusive")
		return 2
	}

	regexText := embeddedRegex
	switch {
	case challenge:
		regexText = embeddedChallengeRegex
	case block:
		regexText = embeddedBlockRegex
	}
	// -D implies -d; the level is "show anything" plus "show the bulky sections".
	debug = debug || debugFull
	var code int
	if url != "" {
		code = runFetch(url, profile, naive, regexText, debug, debugFull, rawOnly, open)
	} else {
		code = runDetect(regexText, debug, debugFull, rawOnly, open)
	}
	maybeNotifyUpdate() // throttled, TTY-only; prints after results, never blocks output
	return code
}

func runDetect(regexText string, debug, full, rawOnly, open bool) int {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: reading stdin: %v\n", err)
		return 2
	}
	return emit(raw, debugContext{fromStdin: true}, regexText, debug, full, rawOnly, open)
}

// runFetch retrieves url directly (browser fingerprint, or Go's default when naive),
// then detects on the captured response chain.
func runFetch(url, profile string, naive bool, regexText string, debug, full, rawOnly, open bool) int {
	raw, err := fetch(url, profile, naive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: %v\n", err)
		return 2
	}
	return emit(raw, debugContext{url: url, profile: profile, naive: naive}, regexText, debug, full, rawOnly, open)
}

// emit writes the chosen output for raw (raw passthrough, debug report, or the
// default slug list), then — when open is set — opens the final response body in
// the user's default browser. It returns the detection exit code.
func emit(raw []byte, ctx debugContext, regexText string, debug, full, rawOnly, open bool) int {
	var code int
	switch {
	case rawOnly:
		os.Stdout.Write(raw)
		code = detect(raw, regexText, true) // exit code only; suppress slug output
	default:
		if debug {
			writeDebug(os.Stdout, raw, ctx, full)
		}
		code = detect(raw, regexText, debug)
	}
	if open {
		if err := openInBrowser(extractFinalBody(raw)); err != nil {
			fmt.Fprintf(os.Stderr, "antibot: opening browser: %v\n", err)
		}
	}
	return code
}

// detect compiles regexText, runs it over raw, prints the sorted vendor slugs
// (unless quiet — the debug report already lists them), and returns the process
// exit code (0 = at least one hit, 1 = none, 2 = bad regex).
func detect(raw []byte, regexText string, quiet bool) int {
	re, err := regexp.Compile(strings.TrimSpace(regexText))
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: embedded regex is invalid: %v\n", err)
		return 2
	}
	hits := Detect(raw, re)
	if !quiet {
		for _, slug := range hits {
			fmt.Println(slug)
		}
	}
	if len(hits) == 0 {
		return 1
	}
	return 0
}
