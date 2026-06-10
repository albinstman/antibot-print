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

Use `-D` to add the full raw response:

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

Update to the latest release:

```console
$ antibot update
antibot: updated abc1234 → def5678
```

> **Tip:** the two fingerprints answer different questions. The default browser fetch
> shows what a real browser gets served. `-n` shows what an obvious bot gets served.
> Some vendors only reveal themselves to one or the other. When in doubt, run both.

## Agent skill

[`SKILL.md`](SKILL.md) is an [agent skill](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview)
that teaches a coding agent (e.g. Claude Code) to use `antibot`. To install it, paste
[`install-skill.md`](install-skill.md) to your agent.

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
