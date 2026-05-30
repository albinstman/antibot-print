# Getting started

If you're already familiar with Kasada and wish to implement the API handling yourself, you can skip this and head directly to the [API Reference](/api-reference/kasada.md).

{% hint style="info" %}
As with any other antibot, make sure you use a working TLS client that mimics the latest version of Google Chrome, match the headers and header-order 1:1 with browser and make sure you are using an up-to-date User-Agent.
{% endhint %}

### Understanding Kasada Flows

Kasada can be implemented in two different ways depending on the website. It's important to identify which flow you're dealing with:

#### Flow 1: Initial Block Page (429 on Homepage)

Some sites, like Hyatt.com, serve a Kasada challenge immediately when you first access the website. You'll receive a **429 status code** with an HTML block page containing a reference to the `ips.js` script.

**Identifying characteristics:**

* First GET request to the website returns 429 status code
* Response body contains: `<script src="/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js?..."></script>`
* You must solve the challenge before accessing any content on the site

**When to use this flow:**

* Site blocks you immediately on homepage access
* You see 429 status code with Kasada script reference
* Site reloads after posting `/tl`

[Read the detailed guide →](/k4sada/flow-1-initial-block-page.md)

#### Flow 2: Fingerprint Endpoint (/fp)

Most sites implement Kasada by having the browser make a request to the `/fp` (fingerprint) endpoint in the background. This is the standard Kasada implementation.

**Identifying characteristics:**

* Browser makes GET request to `/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/fp`
* This request returns 429 with the `ips.js` script reference
* You can often access the site initially, but need to solve Kasada for protected endpoints
* May require `x-kpsdk-cd` header on subsequent requests

**When to use this flow:**

* You can access the homepage but get challenged on specific endpoints
* Browser makes background request to `/fp` endpoint
* You need to maintain `x-kpsdk-ct` token for ongoing requests

[Read the detailed guide →](/k4sada/flow-2-fingerprint-endpoint.md)

### Next Steps

Choose the appropriate flow based on what you observe:

* **Getting 429 immediately on homepage?** → [Flow 1: Initial Block Page](/k4sada/flow-1-initial-block-page.md)
* **Browser making /fp requests?** → [Flow 2: Fingerprint Endpoint](/k4sada/flow-2-fingerprint-endpoint.md)

Both flows share the same core process of fetching the script, generating a payload, and posting to `/tl`. The main difference is when and where the challenge is triggered.

### Complete Example

For a full working implementation with proper TLS client setup, header ordering, and cookie handling, see our examples repository:

{% embed url="<https://github.com/Hyper-Solutions/hypersolutions-examples>" %}


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/k4sada/getting-started.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
