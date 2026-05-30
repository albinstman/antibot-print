# antibot-print — Implementation Plan

## Goal

Produce a **single, highly optimized regular expression** that detects and names the
antibot / WAF vendor(s) protecting a site, from a **static HTTP response only** (status line,
headers, cookies, body) — no JavaScript execution, no headless browser.

The regex is a **build artifact compiled from per-vendor signature files**. One pass over a
normalized response yields every vendor present.

**Output contract (primary):** the regex is **self-describing** — each vendor is a named
capture group, so the names of the groups that fire **are** the answer (a set of vendor
slugs). No external file is required to name vendors. `mapping.json` is an *optional*
side-artifact for consumers who want enriched output (confidence, category, citations).

---

## Locked decisions

| Decision | Choice |
|---|---|
| Regex flavors | Compile **both** RE2-compatible and PCRE from one source; compare at the end (size, speed, correctness parity). |
| Vendor scope | **All vendors with genuine static-HTTP signals.** JS-only fingerprint detectors (canvas, webgl, audio, …) are out of scope — unobservable in a static response. |
| Test data | **Synthesized** from documented signals: positive / negative / cross-vendor / multi-vendor fixtures. Proves internal consistency and no cross-vendor false positives; does **not** prove real-world accuracy (real captures deferred). |
| Architecture | Per-vendor **signature files are the source of truth**; the single regex is generated from them. |
| Output contract | **Vendor-named capture groups** (Rung 3). The names of the groups that match are the output. `mapping.json` is optional enrichment, not required. |
| CLI | **Out of scope.** No bundled executable. Consumers do two steps — normalize, then regex — inline (reference snippets provided). |
| Tooling language | **Python** for the generator/normalizer/tests (best dual-regex ecosystem: `re` + `google-re2`/`pyre2`). Output regex strings are language-agnostic. |

---

## Input model & normalization

### Why normalize at all (the `\r\n\r\n` question)

A raw HTTP/1.1 response on the wire already separates headers from body with `\r\n\r\n`, so in
that single case a regex could anchor to the boundary and skip normalization. We still keep a
thin normalization step because:

- **We rarely have raw wire bytes.** Inputs come from parsed sources (`requests`/`httpx`,
  proxy logs, HAR) that already split the response into `(status, headers, body)`. Using the
  wire format would mean re-serializing back to it — normalization by another name.
- **HTTP/2 and HTTP/3 have no `\r\n` framing** (binary HPACK/QPACK). Most Cloudflare/Akamai/
  DataDome targets are H2/H3. The normalizer gives one uniform input across protocol versions.
- **The wire isn't canonical** — header case, whitespace, cookie joining all vary; normalizing
  lets signatures stay simple and strict.
- **Context tags are the real value** — they let each signal anchor to status/header/body and
  OR cleanly into one alternation.

Normalization is therefore **minimal and wire-aware**: for raw HTTP/1.1 bytes it is nearly a
pass-through (split on first `\r\n\r\n`, lowercase header names, add tags, cap body).

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
- Lines joined with `\n`. The regex is compiled with multiline semantics so `S:`/`H:`/`B:`
  prefixes anchor each signal to its context.

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
- `citation`: source file + exact quote. **Every signal must be cited.**
- `variant` *(optional)*: a sub-label for vendors that expose more than one distinguishable
  challenge/captcha (see "Group naming" below).

## Output contract — vendor-named capture groups (Rung 3)

The compiler produces **one named capture group per vendor**, whose body is an alternation of
that vendor's signals:

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

No mapping file is needed for this — the group *name* is the vendor slug.

### Group naming

- **Default:** group name = vendor slug (`cloudflare`, `datadome`, `aws_waf`). Slug-safe
  (`[a-z0-9_]+`) so it's a valid capture-group name in every engine.
- **Variants (future-proofing):** when a vendor exposes more than one distinguishable
  challenge, set `variant` on the relevant signals and the compiler flattens the group name to
  `<vendor>_<variant>` — e.g. `cloudflare_captcha` vs `cloudflare_interstitial`. The vendor is
  recoverable by splitting on the first `_`. Engines disallow duplicate group names (RE2) so
  each distinct group name appears once, as an alternation of its signals.
- `mapping.json` *(optional)*: `group name → {vendor, display_name, category, confidence,
  citation}` for consumers who want enriched/structured output instead of bare slugs.

---

## Repository structure

```
schema/signature.schema.json      # meta-schema: defines the signature file format
signatures/<vendor>.json          # one per vendor — source of truth
src/normalize.py                  # HTTP response -> canonical tagged string
src/compile.py                    # signatures -> antibot.re2.txt + antibot.pcre.txt (+ mapping.json)
build/antibot.re2.txt             # compiled single regex (RE2)   — PRIMARY deliverable
build/antibot.pcre.txt            # compiled single regex (PCRE)  — PRIMARY deliverable
build/mapping.json                # OPTIONAL: group name -> {vendor, display_name, category, confidence, citation}
tests/fixtures/                   # synthesized labeled responses
tests/test_*.py                   # positive / negative / cross / multi / ReDoS / parity
docs/normalization-spec.md        # the canonical format + reference normalizer snippets (Python/JS/Go/awk)
docs/signal-catalog.md            # human-readable per-vendor signals, every one cited
```

---

## Phased agent plan

| Phase | Work | Agents | Deliverable |
|---|---|---|---|
| **0 — Foundations** | `docs/normalization-spec.md` (canonical format + reference snippets Python/JS/Go/awk) + `signature.schema.json` + repo skeleton + generator/test stubs | 1 (sequential, gates everything) | Schema + normalization contract all downstream agents build against |
| **1 — Reverse-engineer** | One agent **per vendor**: read refs (+ scrapfly detector logic), produce `signatures/<vendor>.json` + `signal-catalog` entry; keep only static-HTTP-observable signals; cite every one | ~10–12 parallel | Per-vendor signature files |
| **1b — Verify** | Adversarial reviewer per signature: each signal genuinely static-HTTP, cited, distinct from other vendors, low false-positive | fan-out, 1 per vendor | Hardened signatures |
| **2 — Compile** | Build `compile.py`; emit RE2 + PCRE (vendor-named groups) + optional `mapping.json`; validate all signatures against schema | 1 | The two single-regex artifacts |
| **3 — Test & compare** | Synthesize fixtures (positive/negative/cross/multi); run both regexes; assert correct groups; ReDoS + speed benchmark; RE2-vs-PCRE comparison | fan-out + synthesis | Test suite + comparison report |
| **4 — Synthesize** | Final report: coverage table, recommended flavor (or ship both), limitations | 1 | Decision doc |

---

## Phase 1 vendor fan-out

- **Documented-deep (from `references/`):** Akamai Bot Manager, DataDome, Incapsula/Imperva, Kasada.
- **HTTP-detectable extras (scrapfly taxonomy, signals to be derived & confidence-marked):**
  Cloudflare, AWS WAF, F5 (Shape/BIG-IP), PerimeterX/HUMAN, Sucuri, Reblaze, ThreatMetrix, CHEQ.
- **Captcha vendors (body/script markers, `category: captcha`):** hCaptcha, reCAPTCHA, FunCaptcha — included where they leave static markers.
- **Excluded:** JS-only fingerprint detectors (canvas, webgl, audio, navigator, …) — not observable in a static response.

Each agent keeps only signals that survive in a static HTTP response and marks confidence.

---

## Definition of done

1. `schema/signature.schema.json` validates every `signatures/*.json`.
2. Each vendor signature has ≥1 `definitive` or `strong` signal, all cited.
3. `compile.py` deterministically emits `antibot.re2.txt`, `antibot.pcre.txt` (vendor-named groups) and the optional `mapping.json`.
4. Test suite passes against the **named-group output (Rung 3)**: each vendor fixture lights up its own group and no other; negatives match nothing; multi-vendor responses report all vendors via their group names.
5. ReDoS check: both regexes run linear-time on adversarial input; speed benchmarked.
6. RE2-vs-PCRE comparison report: parity on all fixtures, plus size/speed and any signal one flavor couldn't express.
7. `docs/signal-catalog.md` documents every vendor's signals with citations.

---

## Decision log / open items

- [x] Regex flavors: both RE2 and PCRE, compared.
- [x] Scope: all HTTP-detectable vendors; JS-only fingerprints excluded.
- [x] Tests: synthesize-only (real captures deferred).
- [x] Build: compile single regex from per-vendor signature files.
- [x] Tooling language: Python.
- [x] Normalization: keep a minimal, wire-aware normalizer (see "Input model").
- [x] Output contract: vendor-named capture groups (Rung 3); group name = vendor slug.
- [x] CLI: out of scope — consumers run normalize + regex inline.
- [x] `mapping.json`: optional enrichment, not required for the primary contract.
- [x] Group naming: `<vendor>`; extensible to `<vendor>_<variant>` for multi-challenge vendors (e.g. `cloudflare_captcha` vs `cloudflare_interstitial`).
- [ ] Body cap size (default 64 KB) — confirm during Phase 0.
- [ ] Real-capture corpus — future work to validate real-world accuracy.
