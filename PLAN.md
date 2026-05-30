# antibot-print — Implementation Plan

## Goal

Produce a **single, highly optimized regular expression** that detects and names the
antibot / WAF vendor(s) protecting a site, from a **static HTTP response only** (status line,
headers, cookies, body) — no JavaScript execution, no headless browser.

The regex is a **build artifact compiled from per-vendor signature files**. One pass over a
normalized response yields every vendor present.

The regex is **self-describing**: each vendor is a named capture group, so the names of the
groups that match are the output (a set of vendor slugs).

---

## Requirements

| Area | Decision |
|---|---|
| Regex engine | **RE2 only.** Author all signatures within RE2's feature set (no backreferences/lookaround). Fall back to PCRE only for a specific signal that is provably impossible to express in RE2, and document why. |
| Vendor scope | All vendors with genuine static-HTTP signals. JS-only fingerprint detectors (canvas, webgl, audio, …) are out of scope — they are unobservable in a static response. |
| Test data | Synthesized from documented signals: positive / negative / cross-vendor / multi-vendor fixtures. This proves internal consistency and the absence of cross-vendor false positives; it does not prove real-world accuracy (a real-capture corpus is future work). |
| Architecture | Per-vendor signature files are the source of truth; the single regex is generated from them. |
| Output | Vendor-named capture groups. The names of the groups that match are the output. |
| Executable | A small `antibot-print` executable reads a raw HTTP response on stdin and prints the detected vendor slug(s) — usable directly in a `curl` pipe. The regex is also usable standalone (normalize + match) without the executable. |
| Tooling language | Python for the generator, normalizer, executable, and tests. The output regex string is language-agnostic. |

---

## Input model & normalization

Before the regex runs, every response is serialized into a single canonical, context-tagged
string. Normalization is minimal and wire-aware: for a raw HTTP/1.1 byte stream it is nearly a
pass-through (split on the first `\r\n\r\n`, lowercase header names, add tags, cap the body).

Normalization gives the regex one uniform input regardless of source:

- Inputs usually arrive already parsed (`requests`/`httpx`, proxy logs, HAR) as
  `(status, headers, body)`, not as raw bytes.
- HTTP/2 and HTTP/3 have no `\r\n` framing (binary HPACK/QPACK), yet cover most major-vendor
  targets; normalization unifies them with HTTP/1.1.
- Raw header case, whitespace, and cookie joining vary; normalizing lets signatures stay simple
  and strict.
- Context tags (`S:`/`H:`/`B:`) let each signal anchor to status, header, or body and OR
  cleanly into one alternation.

### Canonical format (what the regex runs against)

```
S:<status-code>
H:<lowercased-header-name>:<value>          # one line per header
H:set-cookie:<cookie-name>=<value>          # one line per Set-Cookie
B:<body, capped at 64 KB>
```

- Header names lowercased; raw values preserved.
- One line per header and per `Set-Cookie`.
- Body appended under `B:` lines, capped (default 64 KB) to preserve linear-time scanning.
- Lines joined with `\n`. The regex is compiled with multiline semantics so the `S:`/`H:`/`B:`
  prefixes anchor each signal to its context.

The README documents this format and provides reference normalizer snippets (Python, JS, Go,
awk) so any consumer can reproduce the canonical string exactly.

---

## Signature files

One file per vendor at `signatures/<vendor>.json`. A file is a vendor slug plus a list of RE2
patterns; each pattern already includes its `S:`/`H:`/`B:` context prefix:

```json
{
  "vendor": "datadome",
  "signals": [
    "H:set-cookie:\\s*datadome=",
    "H:server:\\s*DataDome"
  ]
}
```

- `vendor`: slug-safe (`[a-z0-9_]+`), used directly as the capture-group name.
- `signals`: non-empty array of RE2-safe patterns, each anchored with a context prefix.

There is no separate JSON Schema file. `compile.py` validates each file inline: `vendor` is a
slug, `signals` is a non-empty array, and every pattern compiles under RE2.

---

## Output: vendor-named capture groups

The compiler produces one named capture group per vendor, whose body is an alternation of that
vendor's signals:

```
(?P<cloudflare>H:server:\s*cloudflare|H:cf-ray:|H:set-cookie:__cf_bm=|B:.*Just a moment)
|(?P<datadome>H:set-cookie:\s*datadome=|H:server:\s*DataDome)
|(?P<akamai>H:set-cookie:\s*_abck=|H:server:\s*AkamaiGHost)
```

Matching globally and collecting the non-null named groups yields the answer directly:

```python
hits = {g for m in regex.finditer(norm) for g, v in m.groupdict().items() if v}
# → {'cloudflare', 'datadome'}
```

The group name is the vendor slug — that is the entire output contract.

---

## Repository structure

```
signatures/<vendor>.json      # source of truth: {vendor, signals:[RE2 patterns]}
src/normalize.py              # raw HTTP response -> canonical tagged string
src/compile.py                # signatures -> build/antibot.re2.txt (also validates signatures)
bin/antibot-print             # executable: raw HTTP on stdin -> vendor slug(s) on stdout
build/antibot.re2.txt         # the compiled single RE2 regex — primary deliverable
tests/fixtures/               # synthesized labeled responses
tests/test_*.py               # positive / negative / cross / multi / ReDoS
README.md                     # created last (see Phase 4)
```

---

## Phased plan

| Phase | Work | Agents | Deliverable |
|---|---|---|---|
| **0 — Foundations** | Repo skeleton; `normalize.py`; `compile.py` (with inline signature validation); `bin/antibot-print` skeleton; test harness stubs. Canonical format is fixed by this plan. | 1 (sequential, gates everything) | Build + normalization contract all downstream work uses |
| **1 — Reverse-engineer** | One agent per vendor: read both reference sources, produce `signatures/<vendor>.json`; keep only static-HTTP-observable signals expressed as RE2 patterns | ~10–12 parallel | Per-vendor signature files |
| **1b — Verify** | Adversarial reviewer per signature: confirm each signal is genuinely static-HTTP, RE2-valid, and distinct from other vendors (low false-positive) | fan-out, 1 per vendor | Hardened signatures |
| **2 — Compile** | `compile.py` emits `build/antibot.re2.txt` (vendor-named groups) and validates all signatures; flag any signal that cannot be expressed in RE2 | 1 | The single regex artifact |
| **3 — Test** | Synthesize fixtures (positive/negative/cross/multi); assert correct named-group output; ReDoS + linear-time + speed checks | fan-out + synthesis | Test suite + report |
| **4 — Package & document** | Finish `bin/antibot-print`; write a simple `README.md`: what this is, repo structure, and per-tool/per-language integration (curl + executable; curl + inline normalize + regex; Python/JS/Go) | 1 | Executable + README |

---

## Reference sources (Phase 1)

Agents read **both** local reference sources:

- `references/hypersolutions-docs/` — vendor challenge documentation: cookies, headers, status codes, script endpoints.
- `references/scrapfly-anti-bot-detector/` — a browser-extension detector. Its `detectors/index.json` and per-detector logic catalog vendors and the signals they emit; extract only the parts observable in a static HTTP response.

---

## Definition of done

1. Every `signatures/*.json` passes `compile.py`'s inline validation.
2. Each vendor has at least one signal.
3. `compile.py` deterministically emits `build/antibot.re2.txt` with vendor-named groups.
4. The test suite passes: each vendor fixture lights up its own group and no other; negative
   fixtures match nothing; multi-vendor responses report all vendors via their group names.
5. ReDoS check: the regex runs in linear time on adversarial input; speed is benchmarked.
6. Any signal that required a PCRE fallback is documented with the reason RE2 was insufficient.
7. `bin/antibot-print` works in a `curl -isS <url> | antibot-print` pipe.
8. `README.md` describes the project, the repo structure, and per-tool/per-language integration.

---

## Configuration & future work

- Body cap size (default 64 KB) — confirm in Phase 0.
- Real-capture corpus — future work to validate real-world accuracy beyond synthesized fixtures.
- PCRE variant — only if a required signal proves inexpressible in RE2.
