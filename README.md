# antibot-print

Fingerprint the antibot / WAF vendor protecting a site using **static HTTP response analysis only** — no JS execution, no headless browser.

Point it at a response and it tells you which antibot vendor (Cloudflare, Akamai, DataDome, PerimeterX, Imperva, etc.) is in front, identified by a single highly optimized regular expression.

## Why

Most antibot detection relies on running JavaScript or driving a real browser. `antibot-print` works purely from the raw HTTP response — status, headers, and body — making it fast, lightweight, and trivial to run at scale.

## Features

- **Static-only** — analyzes the HTTP response, nothing else
- **Vendor identification** — outputs the specific antibot vendor
- **Single optimized regexp** — one pass, one verdict

## Status

🚧 Early development — just getting started.
