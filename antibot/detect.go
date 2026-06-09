// Detection and normalization: turn a raw HTTP response into the canonical,
// context-tagged string the embedded regex runs against, and report the matched
// vendor slugs. This is the runtime half of the project; the regex it uses is the
// precompiled artifact embedded by regex.go (built from signatures/ by package gen).
package antibot

import (
	"regexp"
	"sort"
	"strings"
)

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
