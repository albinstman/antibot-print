package gen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sigDir is the signature source tree, relative to this package's directory.
const sigDir = "../signatures"

// TestCompileSignatures checks the presence tier compiles to valid RE2 and carries
// a named group per vendor.
func TestCompileSignatures(t *testing.T) {
	pattern, err := CompileSignatures(sigDir, "")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if _, err := regexp.Compile(pattern); err != nil {
		t.Fatalf("assembled pattern is not valid RE2: %v", err)
	}
	if !strings.HasPrefix(pattern, "(?m)") {
		t.Errorf("pattern should start with the multiline flag (?m)")
	}
	for _, vendor := range []string{"cloudflare", "akamai", "datadome"} {
		if !strings.Contains(pattern, "(?P<"+vendor+">") {
			t.Errorf("pattern missing named group for %q", vendor)
		}
	}
}

// TestCompileSubsetTiers checks the challenge and block tiers compile to valid RE2
// and are strict subsets (every group's vendor also appears in presence).
func TestCompileSubsetTiers(t *testing.T) {
	presence, err := CompileSignatures(sigDir, "")
	if err != nil {
		t.Fatalf("compile presence: %v", err)
	}
	tiers := map[string]func(dir, outPath string) (string, error){
		"challenge": CompileChallengeSignatures,
		"block":     CompileBlockSignatures,
	}
	groupRe := regexp.MustCompile(`\(\?P<([a-z][a-z0-9_]*)>`)
	for tier, compile := range tiers {
		pattern, err := compile(sigDir, "")
		if err != nil {
			t.Fatalf("compile %s: %v", tier, err)
		}
		if _, err := regexp.Compile(pattern); err != nil {
			t.Fatalf("assembled %s pattern is not valid RE2: %v", tier, err)
		}
		for _, m := range groupRe.FindAllStringSubmatch(pattern, -1) {
			if !strings.Contains(presence, "(?P<"+m[1]+">") {
				t.Errorf("%s vendor %q is not present in the presence tier", tier, m[1])
			}
		}
	}
}

// TestValidateFileRejectsNonSubsetBlock checks a block signal that is not in
// 'signals' is rejected (block, like challenge, must be a subset of presence).
func TestValidateFileRejectsNonSubsetBlock(t *testing.T) {
	dir := t.TempDir()
	sig := `{"vendor":"acme","signals":["H:x:y"],"block":["H:other:z"]}`
	if err := os.WriteFile(filepath.Join(dir, "acme.json"), []byte(sig), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSignatures(dir); err == nil {
		t.Error("expected an error for a block signal outside 'signals', got nil")
	}
}

// TestValidateFileRejectsBadSlug exercises the validation path on a temp signature.
func TestValidateFileRejectsBadSlug(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Bad.json"), []byte(`{"vendor":"Bad","signals":["H:x:y"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSignatures(dir); err == nil {
		t.Error("expected an error for an invalid vendor slug, got nil")
	}
}
