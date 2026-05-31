---
name: detecting-antibot-vendors
description: Print the antibot, WAF, or CAPTCHA vendors protecting a site using the `antibot` CLI. Use when asked what bot protection a site uses or why automated requests get blocked or challenged.
---

# Detecting antibot vendors

`antibot` prints the antibot/WAF/CAPTCHA vendor(s) protecting a site, one slug per line.
Give it a URL and it fetches the site itself with a browser-like fingerprint.

## Prerequisite

Check for the binary: `command -v antibot`. If missing, install it (no sudo; installs to
`~/.local/bin`):

```sh
curl -fsSL https://raw.githubusercontent.com/albinstman/antibot-print/main/install.sh | bash
```

## Usage

Print vendors:

```console
$ antibot https://example.com
cloudflare
```

Add `-c` to report only vendors actively serving a challenge or block, not mere presence:

```console
$ antibot -c https://example.com
cloudflare
```

Add `-n` to fetch with Go's default fingerprint. Surfaces vendors that challenge bots but
pass real browsers:

```console
$ antibot -n https://example.com
perimeterx
```

Use `-p` to impersonate a different browser (e.g. `chrome_133`, `firefox_135`):

```console
$ antibot -p firefox_135 https://example.com
cloudflare
```

`-n` and `-p` are mutually exclusive. Exit status is `0` if any vendor was detected, `1` if
none (a valid result, not a failure), `2` on error.

> **Tip:** the two fingerprints answer different questions. The default browser fetch shows
> what a real browser gets served. `-n` shows what an obvious bot gets served. Some vendors
> only reveal themselves to one or the other. When in doubt, run both and report which fetch
> surfaced each vendor.

## More info

For the full vendor list, signature format, language integration, or implementation details,
see the repo: https://github.com/albinstman/antibot-print
