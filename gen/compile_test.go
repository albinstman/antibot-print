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

// TestCompileChallengeSignatures checks the challenge tier compiles to valid RE2 and
// is a strict subset (every challenge group's vendor also appears in presence).
func TestCompileChallengeSignatures(t *testing.T) {
	presence, err := CompileSignatures(sigDir, "")
	if err != nil {
		t.Fatalf("compile presence: %v", err)
	}
	challenge, err := CompileChallengeSignatures(sigDir, "")
	if err != nil {
		t.Fatalf("compile challenge: %v", err)
	}
	if _, err := regexp.Compile(challenge); err != nil {
		t.Fatalf("assembled challenge pattern is not valid RE2: %v", err)
	}
	groupRe := regexp.MustCompile(`\(\?P<([a-z][a-z0-9_]*)>`)
	for _, m := range groupRe.FindAllStringSubmatch(challenge, -1) {
		if !strings.Contains(presence, "(?P<"+m[1]+">") {
			t.Errorf("challenge vendor %q is not present in the presence tier", m[1])
		}
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
