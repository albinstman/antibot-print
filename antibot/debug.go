// --debug path: instead of just naming vendors, dump a diagnostic of one
// response — how it was fetched, the redirect chain, every vendor matched in the
// active tier with the exact text that triggered it, and (with -D) the normalized
// view the regex runs against plus the full raw response. It prints to stdout in
// place of the slug list.
package antibot

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// debugContext records how the response was obtained, for the debug header.
type debugContext struct {
	fromStdin bool
	url       string
	profile   string
	naive     bool
}

const (
	debugMatchCap = 200  // truncate a single matched span for display
	debugBodyCap  = 2000 // truncate the flattened B: line in the normalized view
)

// vendorMatch is one vendor and the exact normalized substrings its named group
// captured — the "what did it actually see" view powering --debug.
type vendorMatch struct {
	vendor  string
	matched []string
}

// detectVerbose runs re over norm and returns, per vendor (sorted), the exact
// substrings that triggered detection — deduped and sorted. It mirrors Detect's
// match loop but keeps the matched text instead of discarding it.
func detectVerbose(norm string, re *regexp.Regexp) []vendorMatch {
	names := re.SubexpNames()
	hits := map[string]map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(norm, -1) {
		for i, group := range m {
			if i > 0 && group != "" && names[i] != "" {
				if hits[names[i]] == nil {
					hits[names[i]] = map[string]bool{}
				}
				hits[names[i]][group] = true
			}
		}
	}
	vendors := make([]string, 0, len(hits))
	for v := range hits {
		vendors = append(vendors, v)
	}
	sort.Strings(vendors)
	out := make([]vendorMatch, 0, len(vendors))
	for _, v := range vendors {
		texts := make([]string, 0, len(hits[v]))
		for t := range hits[v] {
			texts = append(texts, t)
		}
		sort.Strings(texts)
		out = append(out, vendorMatch{v, texts})
	}
	return out
}

// writeDebug prints the diagnostic for raw to w (stdout in practice, ahead of the
// vendor slugs detect prints). It always reports both tiers — presence and
// challenge — regardless of -c, so the diagnostic shows the full picture: which
// vendors are present and which are actively serving a challenge/block.
//
// The light report (full == false) is the small, console-friendly half: how the
// response was fetched, the status and redirect chain, and every vendor matched
// with the exact text that triggered it. The full report adds the two bulky
// sections — the normalized view and the entire raw response.
func writeDebug(w io.Writer, raw []byte, ctx debugContext, full bool) {
	norm := Normalize(raw, DefaultBodyCap)

	fmt.Fprintln(w, "request:")
	if ctx.fromStdin {
		fmt.Fprintln(w, "  source: stdin (piped response)")
	} else {
		fmt.Fprintf(w, "  url:    %s\n", ctx.url)
		if ctx.naive {
			fmt.Fprintln(w, "  mode:   naive (Go default TLS/HTTP fingerprint)")
		} else {
			fmt.Fprintf(w, "  mode:   browser (profile %s)\n", ctx.profile)
		}
	}

	hops := parseHops(raw)
	status := "(none)"
	if len(hops) > 0 {
		status = strconv.Itoa(hops[len(hops)-1].status)
	}
	fmt.Fprintf(w, "response:\n  status: %s\n  bytes:  %d\n", status, len(raw))
	if len(hops) > 1 {
		writeRedirectChain(w, hops, ctx)
	}

	// Report both tiers, independent of -c: presence (every vendor seen) and
	// challenge (the subset actively challenging/blocking).
	writeDebugTier(w, "presence", norm, embeddedRegex)
	writeDebugTier(w, "challenge", norm, embeddedChallengeRegex)

	if !full {
		// Light report: stop before the bulky sections, but point at how to get them.
		fmt.Fprintln(w, "(run -D/--debug-full for the normalized view + full raw response)")
		return
	}

	fmt.Fprintln(w, "normalized (what the regex matches against):")
	for _, line := range strings.Split(norm, "\n") {
		if strings.HasPrefix(line, "B:") && len(line) > debugBodyCap {
			fmt.Fprintf(w, "  %s… (%d body bytes, truncated)\n", line[:debugBodyCap], len(line)-len("B:"))
			continue
		}
		fmt.Fprintf(w, "  %s\n", line)
	}

	// The full raw response, verbatim, exactly as captured — this is the source of
	// truth Normalize is derived from.
	fmt.Fprintln(w, "raw response:")
	w.Write(raw)
	if n := len(raw); n == 0 || raw[n-1] != '\n' {
		fmt.Fprintln(w)
	}
}

// writeDebugTier prints one detection tier: each matched vendor and the exact
// signal text that triggered it.
func writeDebugTier(w io.Writer, label, norm, regexText string) {
	fmt.Fprintf(w, "detection (%s):\n", label)
	re, err := regexp.Compile(strings.TrimSpace(regexText))
	if err != nil {
		fmt.Fprintf(w, "  <embedded regex invalid: %v>\n", err)
		return
	}
	matches := detectVerbose(norm, re)
	if len(matches) == 0 {
		fmt.Fprintln(w, "  (none)")
		return
	}
	for _, vm := range matches {
		fmt.Fprintf(w, "  %s\n", vm.vendor)
		for _, t := range vm.matched {
			fmt.Fprintf(w, "    ← %s\n", debugTruncate(t, debugMatchCap))
		}
	}
}

// redirectHop is one response in the chain: its status and, for a redirect, the
// Location it pointed at.
type redirectHop struct {
	status   int
	location string
}

// parseHops walks the raw response chain (curl -i -L style: one status-line block
// per hop) and returns each hop's status code and Location header, in order.
func parseHops(raw []byte) []redirectHop {
	text := strings.ReplaceAll(strings.ReplaceAll(latin1(raw), "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(text, "\n")
	var hops []redirectHop
	for i, n := 0, len(lines); i < n; {
		if !isStatusLine.MatchString(lines[i]) {
			i++
			continue
		}
		var h redirectHop
		if m := statusRe.FindStringSubmatch(lines[i]); m != nil {
			h.status, _ = strconv.Atoi(m[1])
		}
		i++
		for i < n && lines[i] != "" { // headers until the blank separator
			if name, val, ok := strings.Cut(lines[i], ":"); ok && strings.EqualFold(strings.TrimSpace(name), "location") {
				h.location = strings.TrimSpace(val)
			}
			i++
		}
		hops = append(hops, h)
		i++                                                // skip blank line
		for i < n && !isStatusLine.MatchString(lines[i]) { // skip body to next hop
			i++
		}
	}
	return hops
}

// writeRedirectChain prints the whole chain on one line as
// "url (status) -> url (status) -> …", each URL beside the status it returned.
// Direct fetches know the URLs (resolving each Location the way the fetch loop
// does); piped input doesn't know the start URL, so that hop shows just "(status)"
// and later hops show the Location targets.
func writeRedirectChain(w io.Writer, hops []redirectHop, ctx debugContext) {
	cur := ctx.url
	if ctx.fromStdin {
		cur = ""
	}
	parts := make([]string, 0, len(hops))
	for _, h := range hops {
		if cur != "" {
			parts = append(parts, fmt.Sprintf("%s (%d)", cur, h.status))
		} else {
			parts = append(parts, fmt.Sprintf("(%d)", h.status))
		}
		// Advance to the URL the next hop will have.
		switch {
		case cur != "" && h.location != "":
			if r, err := resolveLocation(cur, h.location); err == nil {
				cur = r
			} else {
				cur = ""
			}
		case cur == "" && h.location != "": // stdin: next URL is the raw Location
			cur = h.location
		default:
			cur = ""
		}
	}
	fmt.Fprintf(w, "  redirects:\n    %s\n", strings.Join(parts, " -> "))
}

func debugTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
