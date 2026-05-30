# antibot-print

Print the antibot vendors protecting a site by matching its HTTP response against a
single [regex](antibot.re2.txt).

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
> `xattr -d com.apple.quarantine ~/.local/bin/antibot-print`.

## Usage

```console
$ curl -isS https://example.com | antibot-print
cloudflare
```

To run the regex yourself instead of the binary, see [Language integration](#language-integration).

> **Tip:** the tool only sees what's in the response you give it. Evasion-aware WAFs
> (Akamai, DataDome, …) reveal their cookies/scripts only to requests that look like a
> real browser — send browser headers, and ideally a browser TLS fingerprint
> (see [Roadmap](#roadmap)).

## Language integration

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
	if len(body) > 65536 {
		body = body[:65536]
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

def normalize(raw, cap=65536):
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
    req = urllib.request.Request(url, headers={"User-Agent": "antibot-print"})
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

function normalize(raw, cap = 65536) {
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

These normalizers cover the common single-response case; `Normalize` in `main.go` is
the reference (redirect chains, byte handling).

## Project structure

```
signatures/<vendor>.json   source of truth: {vendor, signals:[RE2 patterns]}
main.go                    normalize, compile, detect; embeds the regex
main_test.go               smoke tests + artifact-sync guard
antibot.re2.txt            the compiled regex (embedded in the binary, usable standalone)
install.sh                 curl | bash installer (downloads a release binary)
.github/workflows/release.yml   build 5 platforms on push to main -> rolling "latest" release
```

## Build from source

```sh
go build -o antibot-print .   # embeds antibot.re2.txt
go test ./...                 # smoke tests + artifact-sync check
go run . compile              # regenerate antibot.re2.txt from signatures/
```

To add or change a vendor, edit a `signatures/<vendor>.json` (each signal prefixed
`S:`/`H:`/`B:`, valid RE2, vendor-specific) and run `go run . compile`. Pushing to
`main` recompiles the regex and rebuilds every platform binary into the rolling
**latest** release, so the signature files are the only thing you maintain.

## Roadmap

- [ ] **Fetch the URL directly** — `antibot-print https://example.com` instead of
  piping `curl`, sending a browser-like TLS/HTTP-2 fingerprint and header set so
  evasion-aware WAFs actually reveal themselves. In Go this means an impersonating
  client such as [`bogdanfinn/tls-client`](https://github.com/bogdanfinn/tls-client)
  or [`utls`](https://github.com/refraction-networking/utls); the alternative is to
  keep piping from a TLS-impersonating fetcher
  ([`curl_cffi`](https://github.com/lexiforest/curl_cffi) in Python, or the
  [`lexiforest/curl-impersonate`](https://github.com/lexiforest/curl-impersonate) CLI).
- [ ] **Audit the signals** — keep only signals that indicate an actual antibot
  *challenge*, not mere CDN/vendor presence (e.g. an Akamai-accelerated site not
  running Bot Manager, or generic Google session cookies). Prefer high-confidence,
  challenge-specific signals even when the regex gets gnarly — like the Akamai
  script-endpoint pattern `B:<script[^>]*\bsrc="/(?:[a-zA-Z0-9_-]+/){5,}[a-zA-Z0-9_-]*[A-Z][a-zA-Z0-9_/-]*"`.

## License

[MIT](LICENSE)
