# Flow 2: Fingerprint Endpoint

### Overview

Unlike Flow 1, you may be able to access the homepage initially. Kasada is triggered when the browser makes a GET request to:

```
/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/fp
```

This request returns a 429 status code with the Kasada challenge. After solving it, you'll receive tokens and cookies that must be included in subsequent requests to protected endpoints.

### Step 1: Request the /fp Endpoint

Make a GET request to the fingerprint endpoint with query parameter `x-kpsdk-v`:

{% code overflow="wrap" %}

```
https://www.example.com/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/fp?x-kpsdk-v=j-xxx
```

{% endcode %}

The response will be a **429 status code** with HTML that looks like this:

{% code overflow="wrap" %}

```html
<!DOCTYPE html>
<html>
<head></head>
<body>
<script>window.KPSDK={};KPSDK.now=typeof performance!=='undefined'&&performance.now?performance.now.bind(performance):Date.now.bind(Date);KPSDK.start=KPSDK.now();</script>
<script src="/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js?tkrm_alpekz_s1.3=0ZhprgzXdlDhhn0esTCQPfWjA2AeaGW50gpHSJVGSjRUPSrKJRQmsSZjTK8HhAmopVcLq2dfwum0SJmpM0Kz5j2DupTTI4OB1PLl7lkhhJIVFAKsCsEoeL4hVm2tQjyFkyPUu42RgZ0dutvGd2xxDbpRLCWjV9MlMysNPzGvUTyg8CBX&x-kpsdk-im=AAIHh6ySRFXhFWAJcYSdsr-BStey6j5sKkK9HXfcJJ2BnB2_eCdWiiJjVu0OEOBEhsIFyZ4CgRIcu6EDyMf-WS88HRSC8PKJm2lZpq0ZTummEHy855H_HBuLSiiUmGQSiPUbJ74rXDFbWw"></script>
</body>
</html>
```

{% endcode %}

### Step 2: Parse the Script Path

Extract the script path from the HTML response. The script URL will look like:

```
/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js?...
```

You can parse this using our SDKs:

{% tabs %}
{% tab title="Golang" %}
{% code overflow="wrap" %}

```go
scriptPath, err := kasada.ParseScriptPath(reader)
if err != nil {
    // Handle the error
}
// scriptPath will look like: /149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js?...
```

{% endcode %}
{% endtab %}

{% tab title="Python" %}
{% code overflow="wrap" %}

```python
from hyper_sdk.kasada import parse_script_path

script_path = parse_script_path(blocked_page_html)
# Returns: /ips.js?.
```

{% endcode %}
{% endtab %}

{% tab title="JS / TS" %}
{% code overflow="wrap" %}

```javascript
import { parseKasadaPath } from 'hyper-sdk-js';

const scriptPath = parseKasadaPath(blockedPageHtml);
```

{% endcode %}

{% endtab %}
{% endtabs %}

### Step 3: Fetch the ips.js Script

Make a GET request to the script path you parsed. Make sure to:

* Use the full URL: `https://www.example.com{scriptPath}`
* Match browser headers exactly
* Maintain the same header order as Chrome
* Set referer to the `/fp` URL

Save the JavaScript response body as you'll need it for the next step.

### Step 4: Generate Payload via API

Use the Hyper Solutions API to generate the payload and headers needed for the `/tl` request.\
\
Refer to the [Kasada](/api-reference/kasada.md) and the SDK documentation for accurate fields.

{% tabs %}
{% tab title="Golang" %}
{% code overflow="wrap" %}

```go
payload, headers, err := session.GenerateKasadaPayload(ctx, &hyper.KasadaPayloadInput{
    // Fields
})
if err != nil {
    // Handle the error
}
// payload and headers are ready for the /tl request
```

{% endcode %}

{% endtab %}

{% tab title="Python" %}
{% code overflow="wrap" %}

```python
from hyper_sdk import KasadaPayloadInput

payload, headers = session.generate_kasada_payload(KasadaPayloadInput(
    # kasada payload input fields
))
```

{% endcode %}

{% endtab %}

{% tab title="JS / TS" %}
{% code overflow="wrap" %}

```javascript
import { KasadaPayloadInput, generateKasadaPayload } from 'hyper-sdk-js';

const result = await generateKasadaPayload(session, new KasadaPayloadInput(
    // kasada payload input fields
));

const payload = result.payload;
const headers = result.headers;
```

{% endcode %}

{% endtab %}
{% endtabs %}

{% hint style="warning" %}
The payload returned by the API is base64-encoded. You must decode it before posting to `/tl`.
{% endhint %}

### Step 5: POST to /tl Endpoint

POST the decoded payload to the `/tl` endpoint:

```
https://www.example.com/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/tl
```

**Critical requirements:**

* Content-Type must be `application/octet-stream`
* Include all headers returned by the API (`x-kpsdk-im`, `x-kpsdk-ct`, `x-kpsdk-dt`)
* Match browser header order exactly
* POST the decoded (binary) payload
* Set referer to the `/fp` URL

### Step 6: Parse /tl Response

A successful response will return **200 status code** with:

**Response body:**

```json
{
    "reload": true
}
```

**Critical response headers to save:**

* `x-kpsdk-ct`: Token that must be included in subsequent requests to protected endpoints
* `x-kpsdk-st`: Timestamp value needed for generating POW (`x-kpsdk-cd`) headers
* `set-cookie`: Kasada cookies (e.g., `tkrm_alpekz_s1.3`, `tkrm_alpekz_s1.3-ssn`)

Example response headers:

{% code overflow="wrap" %}

```
x-kpsdk-ct: 02Rrkf95YyBbq2lGyws6SFVp...
x-kpsdk-st: 1759149934586
set-cookie: tkrm_alpekz_s1.3=02Rrkf95YyBbq2lGyws6SFVp...; Max-Age=86400; Path=/; HttpOnly
set-cookie: tkrm_alpekz_s1.3-ssn=02Rrkf95YyBbq2lGyws6SFVp...; Max-Age=86400; Path=/; HttpOnly; Secure; SameSite=None
```

{% endcode %}

{% hint style="info" %}
Store these values in your session:

* Update your cookie jar with the Set-Cookie headers
* Save `x-kpsdk-st` for future POW generation
* Save `x-kpsdk-ct` as you'll need to include it in request headers to protected endpoints
  {% endhint %}

### Step 7: Making Requests to Protected Endpoints

Now you can make requests to protected endpoints with the Kasada tokens. **Observe what headers the browser includes** and match them exactly.

#### Headers to include:

1. **Kasada cookies** (always required):
   * Include all cookies from the `/tl` response in your Cookie header
2. **x-kpsdk-ct header** (if browser includes it):
   * Use the value from the `/tl` response headers, or the one that was last returned from a protected endpoint.
   * Some sites require this in the header, others only use cookies
3. **x-kpsdk-cd header** (if browser includes it):
   * This is a POW (Proof of Work) that must be freshly generated for **each request**
   * See the section below for how to generate it

### Fetch Client Configuration (Optional - /mfc)

Some Kasada implementations require an additional step to fetch client configuration. If you observe the browser making a GET request to the `/mfc` endpoint, you'll need to include this step.

#### When to use /mfc

Check your browser's network logs. If you see a request to:

```
/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/mfc
```

Then you need to perform this step after solving the initial challenge and before making requests to protected endpoints.

#### Requesting /mfc

Make a GET request to the `/mfc` endpoint:

```
https://www.example.com/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/mfc
```

**Important:**

* Include your Kasada cookies from the `/tl` response
* Match browser headers and header order
* The request should return a 200 status code

#### Parse /mfc Response Headers

The response will include these critical headers:

* **x-kpsdk-fc**: Feature configuration value needed for POW generation on sites using `/mfc`
* **x-kpsdk-h**: Header value that may be required on subsequent requests to protected endpoints

Example response headers:

```
x-kpsdk-fc: AH4kT...
x-kpsdk-h: 1-BwVlRFs
```

{% hint style="info" %}
Store both values:

* Save `x-kpsdk-fc` to use when generating POW (`x-kpsdk-cd`) headers
* Save `x-kpsdk-h` and include it in requests to protected endpoints if the browser does
  {% endhint %}

{% hint style="warning" %}
If your site doesn't use `/mfc` (you don't see it in browser logs), you can skip this step entirely and omit the `fc` parameter when generating POW.
{% endhint %}

### Generating x-kpsdk-cd for Each Request

If the website requires the `x-kpsdk-cd` header on requests (check browser behavior), you **must** generate a fresh POW for **each and every request**. Never reuse POW values.

{% tabs %}
{% tab title="Golang" %}
{% code overflow="wrap" %}

```go
powPayload, err := session.GenerateKasadaPow(ctx, &hyper.KasadaPowInput{
    // POW challenge parameters
})
if err != nil {
    // Handle error
}
```

{% endcode %}
{% endtab %}

{% tab title="Python" %}
{% code overflow="wrap" %}

```python
from hyper_sdk import KasadaPowInput

pow_payload = session.generate_kasada_pow(KasadaPowInput(
    # kasada pow input fields
))
```

{% endcode %}

{% endtab %}

{% tab title="JS / TS" %}
{% code overflow="wrap" %}

```javascript
import { KasadaPowInput, generateKasadaPow } from 'hyper-sdk-js';

const powPayload = await generateKasadaPow(session, new KasadaPowInput(
    // kasada pow input fields
));
```

{% endcode %}
{% endtab %}
{% endtabs %}

{% hint style="danger" %}
**CRITICAL**: The `x-kpsdk-cd` header must be regenerated for every single request. Reusing POW values will cause your requests to fail. Generate a new POW immediately before making each request.
{% endhint %}

### When to Re-solve the Challenge

You may need to re-solve the Kasada challenge (repeat the entire flow) if:

* Your Kasada cookies expire
* You receive a 429 response on protected endpoints

To maintain long-running sessions:

* Proactively refresh tokens before they expire
* Handle 429 responses by triggering a new challenge solve

### Summary

The complete flow:

1. ✅ GET request to `/fp` endpoint → Receive 429 with block page
2. ✅ Parse script path from HTML
3. ✅ GET request to ips.js script
4. ✅ Generate payload via Hyper Solutions API
5. ✅ POST decoded payload to /tl endpoint
6. ✅ Parse response headers and cookies
7. ✅ Make requests to protected endpoints with:
   * Kasada cookies (always)
   * `x-kpsdk-ct` header (if browser uses it)
   * `x-kpsdk-cd` header (if browser uses it - generate fresh for each request)

You have now successfully integrated Kasada's flow!


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/k4sada/flow-2-fingerprint-endpoint.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
