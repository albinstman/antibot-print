# Flow 1: Initial Block Page

### Overview

When you make your first GET request to the website, you'll receive a **429 status code** with an HTML response containing a Kasada script reference. You must solve this challenge before you can access any content on the site.

### Initial Request

The response will be a 429 status code with HTML that looks like this:

{% code overflow="wrap" %}

```html
<!DOCTYPE html>
<html>
<head></head>
<body>
<script>window.KPSDK={};KPSDK.now=typeof performance!=='undefined'&&performance.now?performance.now.bind(performance):Date.now.bind(Date);KPSDK.start=KPSDK.now();</script>
<script src="/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js?tkrm_alpekz_s1.3=0ZhprgzXdlDhhn0esTCQPfWjA2AeaGW50gpHSJVGSjRUPSrKJRQmsSZjTK8HhAmopVcLq2dfwum0SJmpM0Kz5j2DupTTI4OB1PLl7dhhJIVFAKsCsEoeL4hVm2tQjyFkyPUu42RgZ0dutvGd2xxDbpRLCWjV9MlMysNPzGvUTyg8CBX&x-kpsdk-im=AAIHh6ySRFXhFWAJcYSdsr-BStey6j5sKkK9HXfcJJ2BnB2_eCdWiiJjVu0OEOBEhsIFyZ4CgRIcu6EDyMf-WS88HRSC8PKJm2lZpq0ZTummEHy855H_HBuLSiiUmGQSiPUbJ74rXDFbWw"></script>
</body>
</html>
```

{% endcode %}

### Step 1: Parse the Script Path

You need to extract the script path from the HTML response. The script URL will look like:

```
/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js?...
```

You can parse this using our SDKs:

{% tabs %}
{% tab title="Go" %}
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

```python
from hyper_sdk.kasada import parse_script_path

script_path = parse_script_path(html_content)
# script_path will look like: /149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js?...
```

{% endtab %}

{% tab title="JS / TS" %}

```javascript
import { parseKasadaPath } from 'hyper-sdk-js';

const scriptPath = parseKasadaPath(blockedPageHtml);
```

{% endtab %}
{% endtabs %}

### Step 2: Fetch the ips.js Script

Make a GET request to the script path you parsed. Make sure to:

* Use the full URL: `https://www.example.com{scriptPath}`
* Match browser headers exactly
* Maintain the same header order as Chrome

Save the JavaScript response body as you'll need it for the next step.

### Step 3: Generate Payload via API

Now you'll use the Hyper Solutions API to generate the payload and headers needed for the `/tl` request.

Refer to the [Kasada](/api-reference/kasada.md) and the SDK documentation for accurate fields.

{% tabs %}
{% tab title="Golang" %}
{% code overflow="wrap" %}

```go
payload, headers, err := session.GenerateKasadaPayload(ctx, &hyper.KasadaPayloadInput{
    // Kasada payload configuration
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

### Step 4: POST to /tl Endpoint

POST the decoded payload to the `/tl` endpoint:

```
https://www.example.com/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/tl
```

**Critical requirements:**

* Content-Type must be `application/octet-stream`
* Include all headers returned by the API (`x-kpsdk-im`, `x-kpsdk-ct`, `x-kpsdk-dt`)
* Match browser header order exactly
* POST the decoded (binary) payload

### Step 5: Parse /tl Response

A successful response will return **200 status code** with:

**Response body:**

```json
{
    "reload": true
}
```

**Critical response headers to save:**

* `x-kpsdk-ct`: Token for subsequent requests (also in cookies)
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
* Save `x-kpsdk-ct` if you need to include it in request headers (check if browser does)
  {% endhint %}

### Step 6: Retry Original Request

Now retry your original request to the website with the Kasada cookies. The site should no longer serve you a 429 block page.

**Make sure to:**

* Include all Kasada cookies in your request
* Maintain proper headers and header order

### Summary

The complete flow:

1. ✅ Initial GET → Receive 429 with block page
2. ✅ Parse script path from HTML
3. ✅ GET request to ips.js script
4. ✅ Generate payload via Hyper Solutions API
5. ✅ POST decoded payload to /tl endpoint
6. ✅ Parse response headers and cookies
7. ✅ Retry original request with cookies

You have now successfully bypassed Kasada's initial block page challenge!


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/k4sada/flow-1-initial-block-page.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
