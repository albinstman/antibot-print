# antibot

Print the antibot vendors protecting a site by matching its HTTP response against a
single [regex](https://github.com/albinstman/antibot-print/releases/tag/latest).

## Install

**macOS, Linux, WSL:**

```sh
curl -fsSL https://raw.githubusercontent.com/albinstman/antibot-print/main/install.sh | bash
```

**Windows** (PowerShell):

```powershell
irm https://raw.githubusercontent.com/albinstman/antibot-print/main/install.ps1 | iex
```

> **macOS:** binaries are unsigned — clear the quarantine flag once with
> `xattr -d com.apple.quarantine ~/.local/bin/antibot`.

## Usage

Print vendors:

```console
$ antibot https://example.com
cloudflare
```

Or pipe response from curl:

```console
$ curl -isS https://example.com | antibot
cloudflare
```

Add `-c` to report only vendors actively serving a challenge or block, not mere
presence:

```console
$ antibot -c https://example.com
cloudflare
```

Add `-n` to fetch with Go's default fingerprint. Surfaces
vendors that challenge bots but pass real browsers:

```console
$ antibot -n https://example.com
perimeterx
```

Use `-p` to impersonate a different browser (e.g. `chrome_146`, `firefox_135`):

```console
$ antibot -p firefox_135 https://example.com
cloudflare
```

Add `-d` for diagnostics:

```console
$ antibot -d https://example.com
request:
  url:    https://example.com
  mode:   browser (profile chrome_148)
response:
  status: 200
  bytes:  4521
  redirects:
    https://example.com (301) -> https://www.example.com/ (200)
detection (presence):
  cloudflare
    ← H:set-cookie:__cf_bm=
detection (challenge):
  (none)
```

Use `-D` to add the normalized view and full raw response:

```console
$ antibot -D https://example.com > debug.txt
```

Use `-r` to only print the raw fetched response:

```console
$ antibot -r https://example.com > response.txt
```

Use `-o` to open the fetched HTML:

```console
$ antibot -o https://example.com
cloudflare
```

To run the regex yourself instead of the binary, see [Language integration](#language-integration).

> **Tip:** the two fingerprints answer different questions. The default browser fetch
> shows what a real browser gets served. `-n` shows what an obvious bot gets served.
> Some vendors only reveal themselves to one or the other. When in doubt, run both.

Update to the latest release:

```console
$ antibot update
antibot: updated abc1234 → def5678
```

## Agent skill

[`SKILL.md`](SKILL.md) is an [agent skill](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview)
that teaches a coding agent (e.g. Claude Code) to use `antibot`. To install it, paste
[`install-skill.md`](install-skill.md) to your agent.

## Language integration

The examples below read `antibot.re2.txt` from the working directory. The compiled
regex is **not** committed to the repo — download it from the
[latest release](https://github.com/albinstman/antibot-print/releases/latest/download/antibot.re2.txt)
(and `antibot-challenge.re2.txt` for the challenge tier), or generate both locally
with `go run ./cmd/gen`.

### Go

```go
package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
)

func normalize(raw []byte) string {
	t := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\r", "\n")
	head, body, _ := strings.Cut(t, "\n\n")
	out := []string{}
	for i, line := range strings.Split(head, "\n") {
		if i == 0 {
			if m := regexp.MustCompile(`HTTP/[\d.]+\s+(\d{3})`).FindStringSubmatch(line); m != nil {
				out = append(out, "S:"+m[1])
			}
			continue
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			out = append(out, "H:"+strings.ToLower(strings.TrimSpace(k))+":"+strings.TrimSpace(v))
		}
	}
	if len(body) > 8*1024*1024 {
		body = body[:8*1024*1024]
	}
	out = append(out, "B:"+strings.NewReplacer("\n", " ", "\t", " ").Replace(body))
	return strings.Join(out, "\n")
}

func main() {
	resp, err := http.Get("https://example.com")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	raw, _ := httputil.DumpResponse(resp, true) // raw status line + headers + body

	regexText, _ := os.ReadFile("antibot.re2.txt")
	re := regexp.MustCompile(strings.TrimSpace(string(regexText)))

	norm := normalize(raw)
	names := re.SubexpNames()
	for _, m := range re.FindAllStringSubmatch(norm, -1) {
		for i, g := range m {
			if i > 0 && g != "" && names[i] != "" {
				fmt.Println(names[i])
			}
		}
	}
}
```

### Python

```python
import re, urllib.request, urllib.error  # or: import re2 as re

def normalize(raw, cap=8 * 1024 * 1024):
    text = raw.decode("latin-1").replace("\r\n", "\n").replace("\r", "\n")
    head, _, body = text.partition("\n\n")
    lines = head.split("\n")
    out = []
    m = re.match(r"HTTP/[\d.]+\s+(\d{3})", lines[0])
    if m:
        out.append("S:" + m.group(1))
    for line in lines[1:]:
        name, sep, val = line.partition(":")
        if sep:
            out.append(f"H:{name.strip().lower()}:{val.strip()}")
    out.append("B:" + body[:cap].replace("\n", " ").replace("\t", " "))
    return "\n".join(out)

def main(url):
    req = urllib.request.Request(url, headers={"User-Agent": "antibot"})
    try:
        r = urllib.request.urlopen(req)
    except urllib.error.HTTPError as e:
        r = e  # a 403/429 challenge is still the response we want
    raw = f"HTTP/1.1 {r.status} {r.reason}\r\n{r.headers}\r\n".encode("latin-1") + r.read()

    rx = re.compile(open("antibot.re2.txt").read().strip())
    norm = normalize(raw)
    for v in sorted({g for m in rx.finditer(norm) for g, val in m.groupdict().items() if val}):
        print(v)

if __name__ == "__main__":
    main("https://example.com")
```

### JavaScript

```js
import fs from "node:fs";

function normalize(raw, cap = 8 * 1024 * 1024) {
  const text = Buffer.from(raw).toString("latin1").replace(/\r\n|\r/g, "\n");
  const i = text.indexOf("\n\n");
  const [head, body] = i < 0 ? [text, ""] : [text.slice(0, i), text.slice(i + 2)];
  const lines = head.split("\n");
  const out = [];
  const m = lines[0].match(/^HTTP\/[\d.]+\s+(\d{3})/);
  if (m) out.push("S:" + m[1]);
  for (const line of lines.slice(1)) {
    const c = line.indexOf(":");
    if (c >= 0) out.push("H:" + line.slice(0, c).trim().toLowerCase() + ":" + line.slice(c + 1).trim());
  }
  out.push("B:" + body.slice(0, cap).replace(/[\r\n\t]/g, " "));
  return out.join("\n");
}

const res = await fetch("https://example.com");
const headers = [...res.headers].map(([k, v]) => `${k}: ${v}`).join("\r\n");
const raw = Buffer.from(`HTTP/1.1 ${res.status} ${res.statusText}\r\n${headers}\r\n\r\n` + (await res.text()));

// JS regex differs from RE2: translate named groups, drop (?m), scope (?i:).
const pat = fs.readFileSync("antibot.re2.txt", "utf8").trim()
  .replace(/^\(\?m\)/, "").replace(/\(\?P</g, "(?<").replace(/\(\?i:/g, "(?:");
const rx = new RegExp(pat, "gmi");

const norm = normalize(raw);
const vendors = new Set();
for (const m of norm.matchAll(rx))
  for (const [name, v] of Object.entries(m.groups || {})) if (v !== undefined) vendors.add(name);
console.log([...vendors].sort());
```

These normalizers cover the common single-response case; `Normalize` in
`antibot/detect.go` is the reference (redirect chains, byte handling).

## Project structure

```
signatures/<vendor>.json       source of truth: {vendor, signals:[RE2], challenge?:[RE2 subset]}

cmd/cli/main.go                CLI entrypoint (package main): forwards to package antibot
cmd/gen/main.go                generator entrypoint (package main): forwards to package gen

antibot/                       package antibot — the CLI library
  run.go                       flags + dispatch
  detect.go                    normalize + detect (the runtime matcher)
  regex.go                     //go:embed of the generated .re2.txt artifacts
  fetch.go                     direct-fetch path: browser-fingerprinted HTTP via tls-client
  debug.go                     `--debug` diagnostic report (request, detection, raw response)
  open.go                      `--open`: extract final response body, open in default browser
  update.go                    `antibot update` + throttled "update available" notifier
  *_test.go                    detect / fetch / debug / update tests
  antibot.re2.txt              GENERATED, gitignored — compiled presence regex (embedded)
  antibot-challenge.re2.txt    GENERATED, gitignored — compiled challenge-only regex (embedded)

gen/                           package gen — compiles signatures/ into the .re2.txt (no embed)
  compile.go                   load/validate signatures, assemble the RE2 artifacts
  compile_test.go              compile / validation tests

install.sh                     curl | bash installer (downloads a release binary)
.github/workflows/release.yml  build 5 platforms on push to main -> rolling "latest" release
vendor/                        vendored Go deps
```

## Build from source

The compiled regex artifacts are **not** committed — generate them first, then build:

```sh
go run ./cmd/gen              # generate antibot/*.re2.txt from signatures/ (required first)
go build -o bin/antibot ./cmd/cli   # embeds the generated artifacts
go test ./...                # detection + compile tests
```

`go run ./cmd/gen` has no embedded files in its import graph, so it works on a clean
checkout — that's how it bootstraps the artifacts the CLI then embeds. You must run it
once after cloning and again after editing any signature.

To add or change a vendor, edit a `signatures/<vendor>.json` (each signal prefixed
`S:`/`H:`/`B:`, valid RE2, vendor-specific) and run `go run ./cmd/gen`. Pushing to
`main` recompiles the regexes, rebuilds every platform binary, and publishes both the
binaries and the compiled `.re2.txt` into the rolling **latest** release — so the
signature files are the only thing you maintain (and the only regex source in git).

## License

[MIT](LICENSE)
