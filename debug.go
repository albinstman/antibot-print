// --debug path: instead of just naming vendors, dump a full diagnostic of one
// response — how it was fetched, the status/redirect chain, every vendor matched
// in BOTH tiers with the exact text that triggered it, the normalized view the
// regex actually runs against, and the full raw response. All of it goes to
// stderr, so stdout stays the clean, pipeable slug list.
package main

import (
	"fmt"
	"io"
	"regexp"
	"sort"
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
// vendor slugs detect prints). It reports the same tier the run uses — challenge
// when challenge is set, presence otherwise — so the diagnostic matches the result.
//
// The light report (full == false) is the small, console-friendly half: how the
// response was fetched, the status chain, and every vendor matched with the exact
// text that triggered it. The full report adds the two bulky sections — the
// normalized view and the entire raw response.
func writeDebug(w io.Writer, raw []byte, ctx debugContext, full, challenge bool) {
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

	chain := "(none)"
	if s := debugStatusChain(norm); len(s) > 0 {
		chain = strings.Join(s, " → ")
	}
	fmt.Fprintf(w, "response:\n  status: %s\n  bytes:  %d\n", chain, len(raw))

	// Report the same tier the run itself uses, so the diagnostic explains the
	// slugs that follow it rather than a different set.
	regexText, label := embeddedRegex, "presence"
	if challenge {
		regexText, label = embeddedChallengeRegex, "challenge"
	}
	writeDebugTier(w, label, norm, regexText)

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

// debugStatusChain pulls the S:<code> lines out of the normalized view, in order,
// so the redirect chain shows as e.g. "301 → 302 → 200".
func debugStatusChain(norm string) []string {
	var out []string
	for _, line := range strings.Split(norm, "\n") {
		if strings.HasPrefix(line, "S:") {
			out = append(out, line[len("S:"):])
		}
	}
	return out
}

func debugTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
