package antibot

import "testing"

func TestAssetName(t *testing.T) {
	for _, c := range []struct {
		goos, goarch, want string
		ok                 bool
	}{
		{"linux", "amd64", "antibot-linux-amd64", true},
		{"linux", "arm64", "antibot-linux-arm64", true},
		{"darwin", "arm64", "antibot-darwin-arm64", true},
		{"windows", "amd64", "antibot-windows-amd64.exe", true},
		{"windows", "arm64", "antibot-windows-amd64.exe", true}, // arm64 win uses amd64
		{"linux", "386", "", false},
		{"plan9", "amd64", "", false},
	} {
		got, err := assetName(c.goos, c.goarch)
		if c.ok && (err != nil || got != c.want) {
			t.Errorf("assetName(%q,%q) = %q,%v; want %q,nil", c.goos, c.goarch, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Errorf("assetName(%q,%q) = %q; want error", c.goos, c.goarch, got)
		}
	}
}

func TestChecksumFor(t *testing.T) {
	sums := "aaaa  antibot-linux-amd64\n" +
		"bbbb *antibot-windows-amd64.exe\n" + // BSD-style '*' marker
		"cccc  SHA256SUMS\n"
	for _, c := range []struct{ asset, want string }{
		{"antibot-linux-amd64", "aaaa"},
		{"antibot-windows-amd64.exe", "bbbb"},
		{"antibot-darwin-arm64", ""}, // absent
	} {
		if got := checksumFor(sums, c.asset); got != c.want {
			t.Errorf("checksumFor(%q) = %q, want %q", c.asset, got, c.want)
		}
	}
}

func TestSHA256Hex(t *testing.T) {
	// echo -n "" | sha256sum
	if got := sha256hex(nil); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("sha256hex(empty) = %q", got)
	}
}
