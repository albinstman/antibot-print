// Self-update: an explicit `antibot update` command plus a throttled, TTY-only
// notifier. The notifier never changes the binary — it prints a one-line stderr hint
// at most once a day and only when stderr is a terminal, so piped/scripted/CI use
// stays silent and offline. `antibot update` is the only thing that replaces the
// binary, and it verifies the download against the release's SHA256SUMS first.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// updateRepo is the source repo for releases; overridable to match the installers.
func updateRepo() string {
	if r := os.Getenv("ANTIBOT_REPO"); r != "" {
		return r
	}
	return "albinstman/antibot-print"
}

func releaseBase() string {
	return "https://github.com/" + updateRepo() + "/releases/latest/download"
}

const (
	checkInterval = 24 * time.Hour
	notifyTimeout = 2 * time.Second
	updateTimeout = 60 * time.Second
	maxBinary     = 200 << 20 // 200 MiB ceiling on a downloaded binary
	maxText       = 1 << 20   // 1 MiB ceiling on VERSION / SHA256SUMS
)

// ---------------------------------------------------------------------------
// Explicit updater: `antibot update`
// ---------------------------------------------------------------------------

func runUpdate() int {
	asset, err := assetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: %v\n", err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	base := releaseBase()

	latest, _ := fetchText(ctx, base+"/VERSION") // best-effort, for messaging
	latest = strings.TrimSpace(latest)
	if latest != "" && latest == version {
		fmt.Fprintf(os.Stderr, "antibot: already up to date (%s)\n", version)
		recordCheck(latest)
		return 0
	}

	bin, err := fetchBytes(ctx, base+"/"+asset, maxBinary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: download failed: %v\n", err)
		return 1
	}
	sums, err := fetchText(ctx, base+"/SHA256SUMS")
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: download failed: %v\n", err)
		return 1
	}
	want := checksumFor(sums, asset)
	if want == "" {
		fmt.Fprintf(os.Stderr, "antibot: no checksum for %s in SHA256SUMS\n", asset)
		return 1
	}
	if got := sha256hex(bin); got != want {
		fmt.Fprintf(os.Stderr, "antibot: checksum mismatch for %s (expected %s, got %s)\n", asset, want, got)
		return 1
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "antibot: cannot locate own binary: %v\n", err)
		return 1
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if err := replaceBinary(exe, bin); err != nil {
		fmt.Fprintf(os.Stderr, "antibot: cannot replace %s: %v\n", exe, err)
		return 1
	}

	to := latest
	if to == "" {
		to = "latest"
	}
	recordCheck(latest)
	fmt.Fprintf(os.Stderr, "antibot: updated %s → %s\n", version, to)
	return 0
}

// replaceBinary atomically swaps the running executable for new bytes. It writes a
// temp file in the same directory (so the rename stays on one filesystem) and renames
// it over the target; on Windows the running exe is moved aside first.
func replaceBinary(exe string, data []byte) error {
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".antibot-update-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		old := exe + ".old"
		os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return err
		}
		if err := os.Rename(tmpName, exe); err != nil {
			os.Rename(old, exe) // roll back
			return err
		}
		os.Remove(old) // best-effort; may be locked while running
		return nil
	}
	return os.Rename(tmpName, exe)
}

// ---------------------------------------------------------------------------
// Passive notifier (throttled, TTY-only, never mutates the binary)
// ---------------------------------------------------------------------------

// maybeNotifyUpdate prints an at-most-daily stderr hint when a newer build exists.
// It is silent for dev builds, when opted out, in CI, or when stderr is not a TTY.
func maybeNotifyUpdate() {
	if version == "dev" || os.Getenv("ANTIBOT_NO_UPDATE_CHECK") != "" || os.Getenv("CI") != "" {
		return
	}
	if !stderrIsTTY() {
		return
	}

	c, _ := readCache()
	notified := false
	if c.Latest != "" && c.Latest != version {
		printUpdateNotice(c.Latest)
		notified = true
	}
	if time.Since(time.Unix(c.CheckedAt, 0)) < checkInterval {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()
	latest, err := fetchText(ctx, releaseBase()+"/VERSION")
	if err != nil {
		return // network/offline: stay silent
	}
	latest = strings.TrimSpace(latest)
	writeCache(updateCache{CheckedAt: time.Now().Unix(), Latest: latest})
	if !notified && latest != "" && latest != version {
		printUpdateNotice(latest)
	}
}

func printUpdateNotice(latest string) {
	fmt.Fprintf(os.Stderr, "antibot: update available (%s → %s) — run 'antibot update'\n", version, latest)
}

func stderrIsTTY() bool {
	fi, err := os.Stderr.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// ---------------------------------------------------------------------------
// Cache: timestamp + last-seen latest version
// ---------------------------------------------------------------------------

type updateCache struct {
	CheckedAt int64  `json:"checked_at"`
	Latest    string `json:"latest"`
}

func cachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "antibot", "update-check.json"), nil
}

func readCache() (updateCache, error) {
	var c updateCache
	p, err := cachePath()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(b, &c)
}

func writeCache(c updateCache) {
	p, err := cachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	if b, err := json.Marshal(c); err == nil {
		os.WriteFile(p, b, 0o644)
	}
}

// recordCheck stamps the cache as freshly checked against latest (used by the updater
// so it doesn't immediately re-notify).
func recordCheck(latest string) {
	if latest != "" {
		writeCache(updateCache{CheckedAt: time.Now().Unix(), Latest: latest})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// assetName returns the release asset for an OS/arch, mirroring the install scripts:
// linux/darwin on amd64/arm64, and windows always amd64 (arm64 runs via emulation).
func assetName(goos, goarch string) (string, error) {
	switch goos {
	case "linux", "darwin":
		if goarch != "amd64" && goarch != "arm64" {
			return "", fmt.Errorf("unsupported architecture %q", goarch)
		}
		return fmt.Sprintf("antibot-%s-%s", goos, goarch), nil
	case "windows":
		return "antibot-windows-amd64.exe", nil
	default:
		return "", fmt.Errorf("unsupported OS %q", goos)
	}
}

// checksumFor returns the lowercase hex checksum for asset from a SHA256SUMS file.
func checksumFor(sums, asset string) string {
	for _, line := range strings.Split(sums, "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		if strings.TrimPrefix(f[1], "*") == asset {
			return strings.ToLower(f[0])
		}
	}
	return ""
}

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func fetchBytes(ctx context.Context, url string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "antibot-updater/"+version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func fetchText(ctx context.Context, url string) (string, error) {
	b, err := fetchBytes(ctx, url, maxText)
	return string(b), err
}
