// Package gen compiles the per-vendor signature files into the single self-describing
// RE2 artifacts the CLI embeds. It is the build-time half of the project, deliberately
// kept free of any //go:embed so it can run on a clean checkout (where the generated
// .txt do not yet exist) to produce them — see cmd/gen.
package gen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Signature is one vendor's source-of-truth file: signatures/<vendor>.json.
// Challenge and Block are subsets of Signals that indicate, beyond mere vendor
// presence, an active challenge being served and a hard block (denied outright,
// nothing to solve); each compiles into its own regex artifact.
type Signature struct {
	Vendor    string   `json:"vendor"`
	Signals   []string `json:"signals"`
	Challenge []string `json:"challenge,omitempty"`
	Block     []string `json:"block,omitempty"`
}

var (
	slugOK        = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	leadingDotRun = regexp.MustCompile(`^\.[*+]\??`) // trim a leading greedy run from body patterns
	sigPrefixes   = []string{"S:", "H:", "B:"}
)

// CompileSignatures validates dir, builds the presence regex (every vendor's full
// signal set), verifies it compiles, and (when outPath != "") writes it. Returns
// the pattern.
func CompileSignatures(dir, outPath string) (string, error) {
	return compileTier(dir, outPath, "presence", func(s Signature) []string { return s.Signals })
}

// CompileChallengeSignatures is CompileSignatures for the challenge-only tier.
func CompileChallengeSignatures(dir, outPath string) (string, error) {
	return compileTier(dir, outPath, "challenge", func(s Signature) []string { return s.Challenge })
}

// CompileBlockSignatures is CompileSignatures for the hard-block-only tier.
func CompileBlockSignatures(dir, outPath string) (string, error) {
	return compileTier(dir, outPath, "block", func(s Signature) []string { return s.Block })
}

// compileTier validates dir, assembles the regex over one tier's signal lists
// (skipping vendors with none — only presence is guaranteed non-empty), verifies
// it compiles, and (when outPath != "") writes it. Returns the pattern.
func compileTier(dir, outPath, tier string, pick func(Signature) []string) (string, error) {
	sigs, err := loadSignatures(dir)
	if err != nil {
		return "", err
	}
	pattern := buildRegex(sigs, pick)
	if _, err := regexp.Compile(pattern); err != nil { // the artifact must itself be valid RE2
		return "", fmt.Errorf("assembled %s regex is invalid: %w", tier, err)
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
	// Every challenge/block entry must be one of the (already-validated) signals, so
	// those tiers are strict subsets of presence and inherit its validation.
	sigSet := make(map[string]bool, len(s.Signals))
	for _, sig := range s.Signals {
		sigSet[sig] = true
	}
	for tier, list := range map[string][]string{"challenge": s.Challenge, "block": s.Block} {
		for _, sig := range list {
			if !sigSet[sig] {
				return s, fmt.Errorf("%s: %s signal %q is not in 'signals' (%s must be a subset)", name, tier, sig, tier)
			}
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

// buildRegex assembles the single multiline RE2 pattern over one tier's signal
// lists, skipping vendors that declare no signals for it.
func buildRegex(sigs []Signature, pick func(Signature) []string) string {
	groups := make([]string, 0, len(sigs))
	for _, s := range sigs {
		if list := pick(s); len(list) > 0 {
			groups = append(groups, vendorGroup(s.Vendor, list))
		}
	}
	return "(?m)" + strings.Join(groups, "|")
}
