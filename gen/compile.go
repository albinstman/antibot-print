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
