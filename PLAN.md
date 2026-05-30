# antibot-print — Implementation Plan

## Goal

Produce a **single, highly optimized regular expression** that detects and names the
antibot / WAF vendor(s) protecting a site, from a **static HTTP response only** (status line,
headers, cookies, body) — no JavaScript execution, no headless browser.

The regex is a **build artifact compiled from per-vendor signature files**. One pass over a
normalized response yields every vendor present.

The regex is **self-describing**: each vendor is a named capture group, so the names of the
groups that match are the output (a set of vendor slugs). No external file is required to name
vendors. An optional `mapping.json` side-artifact provides enriched output (confidence,
category, citations) for consumers that want structured records.

---

## Requirements

| Area | Decision |
|---|---|
| Regex flavors | Compile **both** an RE2-compatible and a PCRE variant from one source, then benchmark them for size, speed, and correctness parity. |
| Vendor scope | All vendors with genuine static-HTTP signals. JS-only fingerprint detectors (canvas, webgl, audio, …) are out of scope — they are unobservable in a static response. |
| Test data | Synthesized from documented signals: positive / negative / cross-vendor / multi-vendor fixtures. This proves internal consistency and the absence of cross-vendor false positives; it does not prove real-world accuracy (a real-capture corpus is future work). |
| Architecture | Per-vendor signature files are the source of truth; the single regex is generated from them. |
| Output contract | Vendor-named capture groups. The names of the groups that match are the output. `mapping.json` is optional enrichment. |
| CLI | Out of scope. No bundled executable. Consumers perform two inline steps — normalize, then run the regex — using the reference snippets provided. |
| Tooling language | Python for the generator, normalizer, and tests (`re` plus `google-re2`/`pyre2` for the two flavors). The output regex strings are language-agnostic. |

---

## Input model & normalization

Before the regex runs, every response is serialized into a single canonical, context-tagged
string. Normalization is minimal and wire-aware: for a raw HTTP/1.1 byte stream it is nearly a
pass-through (split on the first `\r\n\r\n`, lowercase header names, add tags, cap the body).

Normalization exists so the regex has one uniform input regardless of source:

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

A reference normalizer is specified in `docs/normalization-spec.md` with drop-in snippets for
Python, JavaScript, Go, and awk, so any consumer can reproduce the canonical string exactly.

---

## Signature schema

A JSON Schema meta-schema (`schema/signature.schema.json`) defines the format; one data file
per vendor (`signatures/<vendor>.json`) lists its signals.

```json
{
  "vendor": "datadome",
  "display_name": "DataDome",
  "category": "antibot",
  "signals": [
    {
      "id": "datadome_cookie",
      "context": "header",
      "match": "set-cookie:\\s*datadome=",
      "match_type": "regex",
      "confidence": "definitive",
      "citation": {
        "source": "references/hypersolutions-docs/datadome/getting-started.md",
        "quote": "the datadome cookie is set on protected responses"
      }
    }
  ]
}
```

Field reference:

- `context`: `status` | `header` | `body` | `script-url` (script-url is a body sub-context for `<script src=...>` patterns).
- `match_type`: `literal` (escaped before compile) | `regex` (RE2-safe; no backreferences/lookaround).
- `confidence`: `definitive` | `strong` | `weak`.
- `citation`: source file + exact quote. Every signal must be cited.
- `variant` *(optional)*: a sub-label for vendors that expose more than one distinguishable
  challenge/captcha (see "Group naming").

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

No mapping file is needed for this — the group name is the vendor slug.

### Group naming

- **Default:** group name = vendor slug (`cloudflare`, `datadome`, `aws_waf`). Slug-safe
  (`[a-z0-9_]+`) so it is a valid capture-group name in every engine.
- **Variants:** when a vendor exposes more than one distinguishable challenge, set `variant` on
  the relevant signals and the compiler names the group `<vendor>_<variant>` — e.g.
  `cloudflare_captcha` vs `cloudflare_interstitial`. The vendor is recoverable by splitting on
  the first `_`. Each distinct group name appears once (an alternation of its signals), since
  RE2 disallows duplicate group names.
- `mapping.json` *(optional)*: `group name → {vendor, display_name, category, confidence,
  citation}` for consumers that want enriched/structured output instead of bare slugs.

---

## Repository structure

```
schema/signature.schema.json      # meta-schema: defines the signature file format
signatures/<vendor>.json          # one per vendor — source of truth
src/normalize.py                  # HTTP response -> canonical tagged string
src/compile.py                    # signatures -> antibot.re2.txt + antibot.pcre.txt (+ mapping.json)
build/antibot.re2.txt             # compiled single regex (RE2)   — primary deliverable
build/antibot.pcre.txt            # compiled single regex (PCRE)  — primary deliverable
build/mapping.json                # optional: group name -> {vendor, display_name, category, confidence, citation}
tests/fixtures/                   # synthesized labeled responses
tests/test_*.py                   # positive / negative / cross / multi / ReDoS / parity
docs/normalization-spec.md        # canonical format + reference normalizer snippets (Python/JS/Go/awk)
docs/signal-catalog.md            # human-readable per-vendor signals, every one cited
```

---

## Phased plan

| Phase | Work | Agents | Deliverable |
|---|---|---|---|
| **0 — Foundations** | `docs/normalization-spec.md` (canonical format + reference snippets Python/JS/Go/awk) + `signature.schema.json` + repo skeleton + generator/test stubs | 1 (sequential, gates everything) | Schema + normalization contract all downstream work builds against |
| **1 — Reverse-engineer** | One agent per vendor: read the reference material, produce `signatures/<vendor>.json` + a `signal-catalog` entry; keep only static-HTTP-observable signals; cite every one | ~10–12 parallel | Per-vendor signature files |
| **1b — Verify** | Adversarial reviewer per signature: confirm each signal is genuinely static-HTTP, cited, distinct from other vendors, and low false-positive | fan-out, 1 per vendor | Hardened signatures |
| **2 — Compile** | Build `compile.py`; emit RE2 + PCRE (vendor-named groups) + optional `mapping.json`; validate all signatures against the schema | 1 | The two single-regex artifacts |
| **3 — Test & compare** | Synthesize fixtures (positive/negative/cross/multi); run both regexes; assert correct groups; ReDoS + speed benchmark; RE2-vs-PCRE comparison | fan-out + synthesis | Test suite + comparison report |
| **4 — Synthesize** | Final report: coverage table, recommended flavor (or ship both), known limitations | 1 | Decision doc |

---

## Vendor coverage (Phase 1)

- **Documented in `references/`:** Akamai Bot Manager, DataDome, Incapsula/Imperva, Kasada.
- **Additional, signals to be derived and confidence-marked:** Cloudflare, AWS WAF,
  F5 (Shape/BIG-IP), PerimeterX/HUMAN, Sucuri, Reblaze, ThreatMetrix, CHEQ.
- **Captcha (body/script markers, `category: captcha`):** hCaptcha, reCAPTCHA, FunCaptcha —
  included where they leave static markers.
- **Excluded:** JS-only fingerprint detectors (canvas, webgl, audio, navigator, …) — not
  observable in a static response.

Each vendor keeps only signals that survive in a static HTTP response, each marked with a
confidence level.

---

## Definition of done

1. `schema/signature.schema.json` validates every `signatures/*.json`.
2. Each vendor signature has at least one `definitive` or `strong` signal, all cited.
3. `compile.py` deterministically emits `antibot.re2.txt`, `antibot.pcre.txt` (vendor-named
   groups) and the optional `mapping.json`.
4. The test suite passes against the named-group output: each vendor fixture lights up its own
   group and no other; negative fixtures match nothing; multi-vendor responses report all
   vendors via their group names.
5. ReDoS check: both regexes run in linear time on adversarial input; speed is benchmarked.
6. RE2-vs-PCRE comparison report: parity on all fixtures, plus size/speed and any signal one
   flavor could not express.
7. `docs/signal-catalog.md` documents every vendor's signals with citations.

---

## Configuration & future work

- Body cap size (default 64 KB) — confirm in Phase 0.
- Real-capture corpus — future work to validate real-world accuracy beyond synthesized
  fixtures.
