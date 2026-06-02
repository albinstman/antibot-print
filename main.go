// Command antibot names the antibot/WAF/CAPTCHA vendor(s) protecting a site
// from a static HTTP response read on stdin. It is a single self-contained binary:
// the compiled regex is embedded, and Go's stdlib regexp IS RE2 (linear-time, no
// backreferences/lookaround), so nothing else is bundled or required at runtime.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// version is overridden at release time with -ldflags "-X main.version=...".
var version = "dev"

//go:embed antibot.re2.txt
var embeddedRegex string

//go:embed antibot-challenge.re2.txt
var embeddedChallengeRegex string

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
  -c, --challenge   only report vendors actively serving a challenge/block,
                    not mere vendor presence
  -p, --profile P   browser profile to impersonate when fetching a URL (default %s;
                    e.g. chrome_146, firefox_135). chrome_147 and chrome_148 are
                    synthesized from chrome_146's fingerprint with a bumped User-Agent
  -n, --naive       fetch with Go's default (non-browser) TLS/HTTP fingerprint
                    instead of impersonating a browser — surfaces vendors that
                    challenge suspicious clients but pass real browsers silently
  -d, --debug       print a light diagnostic instead of the slug list: how the
                    response was fetched, the status chain, and every vendor
                    matched in the active tier with the exact text that triggered
                    it (respects -c)
  -D, --debug-full  like --debug, plus the two bulky sections — the normalized
                    view the regex runs against and the full raw response;
                    best redirected to a file (antibot -D URL > debug.txt)
  -r, --raw         print only the raw fetched response (status line, headers,
                    body — like 'curl -i -L'), no detection output; the exit code
                    still reflects detection (0 vendor found, 1 none)
  -h, --help        show this help and exit
  -V, --version     show version and exit

Commands:
  update            download and install the latest release (verifies checksum)
  compile [--dir signatures] [--out antibot.re2.txt]
          [--challenge-out antibot-challenge.re2.txt]
                    regenerate the regex artifacts from signature files (dev/CI)

Environment:
  ANTIBOT_NO_UPDATE_CHECK   disable the daily "update available" check
`

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "compile":
			os.Exit(runCompile(args[1:]))
		case "update":
			os.Exit(runUpdate())
		}
	}

	challenge := false
	naive := false
	debug := false
	debugFull := false
	rawOnly := false
	profile := defaultProfile
	profileSet := false
	url := ""
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-h" || a == "--help":
			fmt.Printf(usage, defaultProfile)
			return
		case a == "-V" || a == "--version":
			fmt.Printf("antibot %s\n", version)
			return
		case a == "-c" || a == "--challenge":
			challenge = true
		case a == "-n" || a == "--naive":
			naive = true
		case a == "-d" || a == "--debug":
			debug = true
		case a == "-D" || a == "--debug-full":
			debugFull = true
		case a == "-r" || a == "--raw":
			rawOnly = true
		case a == "-p" || a == "--profile":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "antibot: --profile requires a value")
				os.Exit(2)
			}
			profile = args[i]
			profileSet = true
		case strings.HasPrefix(a, "-") && a != "-":
			fmt.Fprintf(os.Stderr, "antibot: unknown option %q (try --help)\n", a)
			os.Exit(2)
		default:
			if url != "" {
				fmt.Fprintf(os.Stderr, "antibot: unexpected extra argument %q\n", a)
				os.Exit(2)
			}
			url = a
		}
	}
	if naive && profileSet {
		fmt.Fprintln(os.Stderr, "antibot: --naive and --profile are mutually exclusive")
		os.Exit(2)
	}

	regexText := embeddedRegex
	if challenge {
		regexText = embeddedChallengeRegex
	}
	// -D implies -d; the level is "show anything" plus "show the bulky sections".
	debug = debug || debugFull
	var code int
	if url != "" {
		code = runFetch(url, profile, naive, regexText, debug, debugFull, challenge, rawOnly)
	} else {
		code = runDetect(regexText, debug, debugFull, challenge, rawOnly)
	}
	maybeNotifyUpdate() // throttled, TTY-only; prints after results, never blocks output
	os.Exit(code)
}

func runDetect(regexText string, debug, full, challenge, rawOnly bool) int {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: reading stdin: %v\n", err)
		return 2
	}
	if rawOnly {
		os.Stdout.Write(raw)
		return detect(raw, regexText, true) // exit code only; suppress slug output
	}
	if debug {
		writeDebug(os.Stdout, raw, debugContext{fromStdin: true}, full, challenge)
	}
	return detect(raw, regexText, debug)
}

// runFetch retrieves url directly (browser fingerprint, or Go's default when naive),
// then detects on the captured response chain.
func runFetch(url, profile string, naive bool, regexText string, debug, full, challenge, rawOnly bool) int {
	raw, err := fetch(url, profile, naive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: %v\n", err)
		return 2
	}
	if rawOnly {
		os.Stdout.Write(raw)
		return detect(raw, regexText, true) // exit code only; suppress slug output
	}
	if debug {
		writeDebug(os.Stdout, raw, debugContext{url: url, profile: profile, naive: naive}, full, challenge)
	}
	return detect(raw, regexText, debug)
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

func runCompile(args []string) int {
	dir, out, chOut := "signatures", "antibot.re2.txt", "antibot-challenge.re2.txt"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			i++
			if i < len(args) {
				dir = args[i]
			}
		case "--out":
			i++
			if i < len(args) {
				out = args[i]
			}
		case "--challenge-out":
			i++
			if i < len(args) {
				chOut = args[i]
			}
		default:
			fmt.Fprintf(os.Stderr, "compile: unknown argument %q\n", args[i])
			return 2
		}
	}
	pattern, err := CompileSignatures(dir, out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "compile: wrote %s (%d bytes)\n", out, len(pattern))
	chPattern, err := CompileChallengeSignatures(dir, chOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "compile: wrote %s (%d bytes)\n", chOut, len(chPattern))
	return 0
}

// ---------------------------------------------------------------------------
// Detection
// ---------------------------------------------------------------------------

// Detect normalizes a raw HTTP response and returns the sorted vendor slugs whose
// named capture groups matched.
func Detect(raw []byte, re *regexp.Regexp) []string {
	norm := Normalize(raw, DefaultBodyCap)
	names := re.SubexpNames()
	seen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(norm, -1) {
		for i, group := range m {
			if i > 0 && group != "" && names[i] != "" {
				seen[names[i]] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Normalization: raw HTTP response -> canonical, context-tagged string
//
//	S:<status-code>                     one line per response status
//	H:<lowercased-header-name>:<value>  one line per header / Set-Cookie
//	B:<body, newlines flattened, capped at DefaultBodyCap>
// ---------------------------------------------------------------------------

// DefaultBodyCap bounds how much of the body the regex scans. Go's regexp is RE2
// (linear-time) and fetch already holds the whole body in memory, so this is set
// generously — high enough to reach widgets/scripts planted deep in large pages —
// while still capping work on a pathologically large response.
const DefaultBodyCap = 8 * 1024 * 1024 // 8 MB

var (
	statusRe      = regexp.MustCompile(`HTTP/[\d.]+\s+(\d{3})`)
	isStatusLine  = regexp.MustCompile(`^HTTP/\d`)
	stripBodyChar = strings.NewReplacer("\n", " ", "\t", " ")
)

// Normalize serializes a raw HTTP response into the canonical string. Bytes are
// read as latin-1 (1 byte -> 1 rune) so the engine sees valid UTF-8 with 1:1 byte
// semantics regardless of the body's real charset.
func Normalize(raw []byte, bodyCap int) string {
	text := latin1(raw)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")

	var statuses, headers, bodyLines []string

	idx, n, sawBlock := 0, len(lines), false
	for idx < n {
		if isStatusLine.MatchString(lines[idx]) {
			sawBlock = true
			if m := statusRe.FindStringSubmatch(lines[idx]); m != nil {
				statuses = append(statuses, "S:"+m[1])
			}
			idx++
			for idx < n && lines[idx] != "" { // header lines until blank separator
				if name, val, ok := strings.Cut(lines[idx], ":"); ok {
					headers = append(headers, "H:"+strings.ToLower(strings.TrimSpace(name))+":"+strings.TrimSpace(val))
				}
				idx++
			}
			idx++ // skip blank line
			start := idx
			for idx < n && !isStatusLine.MatchString(lines[idx]) {
				idx++
			}
			bodyLines = lines[start:idx] // keep the final block's body (-L chains)
		} else {
			if !sawBlock {
				bodyLines = lines[idx:] // bare body, no status line
			}
			break
		}
	}

	out := make([]string, 0, len(statuses)+len(headers)+1)
	out = append(out, statuses...)
	out = append(out, headers...)

	body := strings.Join(bodyLines, "\n")
	// Cap by rune count: latin1 made 1 rune == 1 original byte, so this caps the
	// body at bodyCap original bytes (matching the reference normalizer).
	if r := []rune(body); len(r) > bodyCap {
		body = string(r[:bodyCap])
	}
	out = append(out, "B:"+stripBodyChar.Replace(body))
	return strings.Join(out, "\n")
}

// latin1 maps each input byte to the rune of the same value, yielding valid UTF-8.
func latin1(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b))
	for _, c := range b {
		sb.WriteRune(rune(c))
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// Signatures: per-vendor source files -> the single self-describing regex
// ---------------------------------------------------------------------------

// Signature is one vendor's source-of-truth file: signatures/<vendor>.json.
// Challenge is the subset of Signals that indicates an active challenge/block
// (not mere vendor presence); it compiles into a separate regex artifact.
type Signature struct {
	Vendor    string   `json:"vendor"`
	Signals   []string `json:"signals"`
	Challenge []string `json:"challenge,omitempty"`
}

var (
	slugOK        = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	leadingDotRun = regexp.MustCompile(`^\.[*+]\??`) // trim a leading greedy run from body patterns
	sigPrefixes   = []string{"S:", "H:", "B:"}
)

// CompileSignatures validates dir, builds the regex, verifies it compiles, and
// (when outPath != "") writes it. Returns the pattern.
func CompileSignatures(dir, outPath string) (string, error) {
	sigs, err := loadSignatures(dir)
	if err != nil {
		return "", err
	}
	pattern := buildRegex(sigs)
	if _, err := regexp.Compile(pattern); err != nil { // the artifact must itself be valid RE2
		return "", fmt.Errorf("assembled regex is invalid: %w", err)
	}
	if outPath != "" {
		if err := os.WriteFile(outPath, []byte(pattern+"\n"), 0o644); err != nil {
			return "", err
		}
	}
	return pattern, nil
}

// CompileChallengeSignatures is CompileSignatures for the challenge-only tier:
// it validates dir, builds the challenge-subset regex, verifies it, and (when
// outPath != "") writes it. Returns the pattern.
func CompileChallengeSignatures(dir, outPath string) (string, error) {
	sigs, err := loadSignatures(dir)
	if err != nil {
		return "", err
	}
	pattern := buildChallengeRegex(sigs)
	if _, err := regexp.Compile(pattern); err != nil { // the artifact must itself be valid RE2
		return "", fmt.Errorf("assembled challenge regex is invalid: %w", err)
	}
	if outPath != "" {
		if err := os.WriteFile(outPath, []byte(pattern+"\n"), 0o644); err != nil {
			return "", err
		}
	}
	return pattern, nil
}

// loadSignatures validates every *.json in dir and returns them sorted by vendor.
func loadSignatures(dir string) ([]Signature, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no signature files found in %s", dir)
	}
	seen := map[string]bool{}
	sigs := make([]Signature, 0, len(paths))
	for _, p := range paths {
		s, err := validateFile(p)
		if err != nil {
			return nil, err
		}
		if seen[s.Vendor] {
			return nil, fmt.Errorf("duplicate vendor slug: %s", s.Vendor)
		}
		seen[s.Vendor] = true
		sigs = append(sigs, s)
	}
	sort.Slice(sigs, func(i, j int) bool { return sigs[i].Vendor < sigs[j].Vendor })
	return sigs, nil
}

func validateFile(path string) (Signature, error) {
	var s Signature
	raw, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	name := filepath.Base(path)
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("%s: invalid JSON: %w", name, err)
	}
	if !slugOK.MatchString(s.Vendor) {
		return s, fmt.Errorf("%s: 'vendor' must match [a-z][a-z0-9_]* (got %q)", name, s.Vendor)
	}
	if stem := strings.TrimSuffix(name, ".json"); stem != s.Vendor {
		return s, fmt.Errorf("%s: filename stem must equal vendor slug %q", name, s.Vendor)
	}
	if len(s.Signals) == 0 {
		return s, fmt.Errorf("%s: 'signals' must be a non-empty array", name)
	}
	for _, sig := range s.Signals {
		if sig == "" {
			return s, fmt.Errorf("%s: every signal must be a non-empty string", name)
		}
		if !hasSigPrefix(sig) {
			return s, fmt.Errorf("%s: signal must start with S:/H:/B: : %q", name, sig)
		}
		if _, err := regexp.Compile(sig); err != nil { // Go's regexp is RE2: rejects backrefs/lookaround
			return s, fmt.Errorf("%s: signal is not valid RE2: %q: %w", name, sig, err)
		}
	}
	// Every challenge entry must be one of the (already-validated) signals, so the
	// challenge tier is a strict subset of presence and inherits its validation.
	sigSet := make(map[string]bool, len(s.Signals))
	for _, sig := range s.Signals {
		sigSet[sig] = true
	}
	for _, sig := range s.Challenge {
		if !sigSet[sig] {
			return s, fmt.Errorf("%s: challenge signal %q is not in 'signals' (challenge must be a subset)", name, sig)
		}
	}
	return s, nil
}

func hasSigPrefix(sig string) bool {
	for _, p := range sigPrefixes {
		if strings.HasPrefix(sig, p) {
			return true
		}
	}
	return false
}

// vendorGroup builds one named alternation group for a vendor's pattern list.
// S:/H: signals are line-anchored for context precision; B: signals match a
// minimal span anywhere so one finditer pass reports every body-only vendor.
func vendorGroup(vendor string, sigs []string) string {
	alts := make([]string, len(sigs))
	for j, sig := range sigs {
		if strings.HasPrefix(sig, "B:") {
			alts[j] = "(?:" + leadingDotRun.ReplaceAllString(sig[2:], "") + ")"
		} else {
			alts[j] = "^(?:" + sig + ")"
		}
	}
	return "(?P<" + vendor + ">" + strings.Join(alts, "|") + ")"
}

// buildRegex assembles the single multiline RE2 pattern over every vendor's full
// presence signal set.
func buildRegex(sigs []Signature) string {
	groups := make([]string, len(sigs))
	for i, s := range sigs {
		groups[i] = vendorGroup(s.Vendor, s.Signals)
	}
	return "(?m)" + strings.Join(groups, "|")
}

// buildChallengeRegex assembles the multiline RE2 pattern over only the challenge
// subset, skipping vendors that declare no challenge signals.
func buildChallengeRegex(sigs []Signature) string {
	groups := make([]string, 0, len(sigs))
	for _, s := range sigs {
		if len(s.Challenge) == 0 {
			continue
		}
		groups = append(groups, vendorGroup(s.Vendor, s.Challenge))
	}
	return "(?m)" + strings.Join(groups, "|")
}
