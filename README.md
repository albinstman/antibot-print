# antibot-print

Fingerprint the antibot / WAF vendor protecting a site using **static HTTP response analysis only** — no JS execution, no headless browser.

Point it at an HTTP response (status, headers, cookies, body) and it tells you which antibot vendor is in front — Akamai Bot Manager, DataDome, Imperva/Incapsula, Kasada, Cloudflare, PerimeterX, F5, AWS WAF, etc. — identified by a single highly optimized regular expression.

## Why

Most antibot detection relies on running JavaScript or driving a real browser. `antibot-print` works purely from the raw HTTP response, making it fast, lightweight, and trivial to run at scale.

## Headline features

- **Static-only** — analyzes the HTTP response, nothing else
- **Vendor identification** — outputs the specific antibot vendor
- **Single optimized regexp** — one pass over the response, one verdict

## Status

🚧 Early development. We are currently in the **reverse-engineering / research phase**: studying how each vendor manifests in a static HTTP response (block pages, challenge cookies, headers, script endpoints, status codes) so we can derive precise, low-false-positive signatures.

---

## Repository layout

```
antibot-print/
├── README.md            # this file
├── .gitignore
├── references/          # local-only research material (gitignored)
└── scrapfly-anti-bot-detector/   # local-only reference extension (gitignored)
```

> **Note:** `references/` and `scrapfly-anti-bot-detector/` are **gitignored** — they are
> third-party material kept locally for research and are intentionally not published to
> this public repo. Everything below describes what exists on disk for the
> reverse-engineering agent to read.

### `references/hypersolutions-docs/` — vendor challenge documentation

Markdown export of the [Hyper Solutions docs](https://docs.hypersolutions.co/), covering the
internal mechanics, challenge flows, status codes, cookies, and script endpoints of four
major antibot vendors. These are the primary source for deriving static signatures.

| Vendor | Page | What it documents |
|---|---|---|
| **Akamai** | `akamai-web/getting-started.md` | Sensor data flow, `_abck` cookie, script endpoint |
| **Akamai** | `akamai-web/sbsd-introduction.md` | SBSD (server-based sensor data) overview |
| **Akamai** | `akamai-web/sbsd-challenge-flow.md` | SBSD challenge request/response flow |
| **Akamai** | `akamai-web/handling-429-status-codes-with-sbsd-challenges.md` | 429 rate-limit handling under SBSD |
| **Akamai** | `akamai-web/handling-428-status-code-sec-cpt.md` | 428 status + `sec-cpt` challenge |
| **Incapsula** | `incapsula/getting-started.md` | Imperva/Incapsula overview |
| **Incapsula** | `incapsula/reese84.md` | `reese84` interrogation cookie/token |
| **Incapsula** | `incapsula/reese84-dynamic.md` | Dynamic `reese84` variant |
| **Incapsula** | `incapsula/incapsula-captcha-block.md` | Captcha block page behavior |
| **Incapsula** | `incapsula/utmvc.md` | `utmvc` / `___utmvc` cookie challenge |
| **DataDome** | `datadome/getting-started.md` | DataDome overview, cookies, headers |
| **DataDome** | `datadome/tags.md` | DataDome tags / payload |
| **DataDome** | `datadome/slider-captcha.md` | Slider captcha flow |
| **DataDome** | `datadome/interstitial.md` | Interstitial challenge flow |
| **Kasada (k4sada)** | `k4sada/getting-started.md` | Kasada overview |
| **Kasada (k4sada)** | `k4sada/flow-1-initial-block-page.md` | Initial block page (147 status, `x-kpsdk-*`) |
| **Kasada (k4sada)** | `k4sada/flow-2-fingerprint-endpoint.md` | Fingerprint POST endpoint |
| **Kasada (k4sada)** | `k4sada/vercel-botid.md` | Vercel BotID (Kasada-powered) |
| **Kasada (k4sada)** | `k4sada/supported-user-agents.md` | Supported UA list |

### `scrapfly-anti-bot-detector/` — reference browser extension

The unpacked [Scrapfly anti-bot detector](https://scrapfly.io/) Chrome extension, kept as a
reference implementation. It is a **runtime/in-browser** detector (it hooks JS APIs and
inspects the live page), so it is the *opposite* approach to ours — but its
`detectors/index.json` and per-detector logic are a useful catalog of vendors and the
signals they look for.

Vendors it recognizes (`detectors/index.json`):

- **Antibot:** Akamai, Cloudflare, AWS WAF, BotGuard, F5, DataDome, Incapsula, PerimeterX, Shape Security, Sucuri, Reblaze, ThreatMetrix, Meetrics, Ocule, CHEQ, Kasada
- **Captcha:** hCaptcha, reCAPTCHA, GeeTest, QCloud, FunCaptcha, AliExpress, FriendlyCaptcha, Captcha.eu
- **Fingerprint:** audio, canvas, WebGL, WebRTC, font, navigator, screen, timezone, and ~15 more

> Useful for: the vendor taxonomy and the *kinds* of signals each vendor emits. Not directly
> reusable, since it relies on JS instrumentation rather than static HTTP analysis.

---

## Next step

Run a reverse-engineering pass over `references/` to extract, per vendor, the **static HTTP
signals** observable without executing JavaScript:

- distinctive response/challenge **status codes** (e.g. Akamai 428/429, Kasada 429/147)
- **cookies** set on block/challenge (`_abck`, `reese84`, `___utmvc`, `datadome`, `x-kpsdk-*`)
- **response headers** unique to each vendor
- **block/challenge page body** markers and script-endpoint URL patterns

…then consolidate them into the single optimized regexp that powers `antibot-print`.
