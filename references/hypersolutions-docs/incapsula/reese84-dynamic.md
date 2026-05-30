# Reese84 Dynamic

#### Challenge Flow Overview

The Reese challenge follows this sequence:

1. **Initial Request**: Client requests a protected page and receives a challenge page
2. **Extract Script URL**: Parse the challenge page HTML to find the Reese84 script path
3. **Fetch Script Content**: GET the script URL and store the full response body (required by the API)
4. **PoW Request** (if required): Client retrieves the Proof of Work value
5. **Payload Generation**: Client uses our API to generate the correct payload
6. **Payload Submission**: Client submits the payload to the challenge endpoint
7. **Access Granted**: Upon verification, client can access the protected content

#### Implementation Guide

**Step 1: Initial Request & Challenge Detection**

When you make a request to a protected resource, you'll receive a "Pardon Our Interruption" page instead of the expected content:

```http
GET / HTTP/2
Chrome: Headers
```

The response will contain HTML with a script tag that includes essential parameters:

```html
<script>
  if (!isSpa) {
    var scriptElement = document.createElement('script');
    scriptElement.type = "text/javascript";
    scriptElement.src = "/onalbaine-legeance-what-come-Womany-Malcome-to-o/14167535692918208311?s=xcUvM9nI";
    scriptElement.async = true;
    scriptElement.defer = true;
    document.head.appendChild(scriptElement);
  }
</script>
```

**Step 2: Extract Script URL**

Extract the **script path** and **full URL** from the challenge page. You need two values:

* **Script path** (without query params) — used later for the PoW and payload submission endpoints
* **Full script path** (with query params) — used to fetch the script content

```javascript
// Extract the script path (without query params) for the POST endpoints
const pathRegex = /src\s*=\s*"(\/[^/]+\/[^?]+)\?.*"/;
const pathMatches = pathRegex.exec(htmlContent);
const scriptPath = pathMatches[1];
// e.g., "/onalbaine-legeance-what-come-Womany-Malcome-to-o/14167535692918208311"

// Extract the full script path (with query params) for fetching
const fullPathRegex = /scriptElement\.src\s*=\s*"(.*?)"/;
const fullMatches = fullPathRegex.exec(htmlContent);
const fullScriptPath = fullMatches[1];
// e.g., "/onalbaine-legeance-what-come-Womany-Malcome-to-o/14167535692918208311?s=xcUvM9nI"

const scriptUrl = `https://www.example.com${fullScriptPath}`;
```

**Step 3: Fetch & Store the Script Content**

**This step is critical.** Make a GET request to the full script URL and save the entire response body. The script content is required by our API to generate a valid payload.

```http
GET /onalbaine-legeance-what-come-Womany-Malcome-to-o/14167535692918208311?s=xcUvM9nI HTTP/2
Chrome: Headers
Accept: */*
Sec-Fetch-Dest: script
Sec-Fetch-Mode: no-cors
Sec-Fetch-Site: same-origin
Referer: https://www.example.com/
```

Store the full response body as a string — you will pass it to the API in Step 5.

**Step 4: Retrieve Proof of Work (If Required)**

Some sites require an additional Proof of Work (PoW) challenge. To determine if a site requires PoW, observe the network requests in your browser's developer tools. If you see a POST request to the Reese84 script endpoint with the body `{"f":"gpc"}`, the site uses PoW.

If PoW is required, make a POST request to the script path with `?d=yourdomain.com` appended:

```http
POST /onalbaine-legeance-what-come-Womany-Malcome-to-o/14167535692918208311?d=www.example.com HTTP/2
Chrome: Headers
Content-Type: text/plain; charset=utf-8

{"f":"gpc"}
```

The server will respond with a PoW string value:

```json
"eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..."
```

Save this value for the next step.

**Step 5: Generate the Payload**

Use our API to generate the Reese payload. The API requires the following inputs:

| Parameter        | Description                                                                     |
| ---------------- | ------------------------------------------------------------------------------- |
| `UserAgent`      | Your browser's User-Agent string (must be consistent across all requests)       |
| `AcceptLanguage` | Your Accept-Language header value (e.g., `en-US,en;q=0.9`)                      |
| `IP`             | Your client's public IP address                                                 |
| `ScriptUrl`      | The full script URL from Step 2 (e.g., `https://www.example.com/path/id?s=...`) |
| `PageUrl`        | The URL of the protected page you're trying to access                           |
| `Script`         | The full script content fetched in Step 3                                       |
| `Pow`            | The PoW value from Step 4 (empty string if PoW is not required)                 |

Our API will return the properly formatted payload string needed for the next step.

For detailed API specifications, see our [API Reference Documentation](https://claude.ai/api-reference/incapsula.md).

**Step 6: Submit the Payload**

Post the generated payload to the challenge endpoint. Note: this uses the **script path** (without query params) with `?d=yourdomain.com` appended:

```http
POST /onalbaine-legeance-what-come-Womany-Malcome-to-o/14167535692918208311?d=www.example.com HTTP/2
Chrome: Headers
Content-Type: text/plain; charset=utf-8
Accept: application/json; charset=utf-8
Origin: https://www.example.com
Referer: https://www.example.com/

[YOUR_GENERATED_PAYLOAD]
```

The server will respond with a token in JSON format:

```json
{
  "token": "3:2wlemniq+CXN97167oNjyw==:EraPjamz...",
  "renewInSec": 896,
  "cookieDomain": "www.example.com"
}
```

**Step 7: Store the Token & Access Protected Content**

Save the token as a cookie named `reese84` with the domain from the response. Now make your request to the previously protected resource:

```http
GET / HTTP/2
Chrome: Headers
Cookie: reese84=3:2wlemniq+CXN97167oNjyw==:EraPjamz...
```

Verify the response no longer contains the "Pardon Our Interruption" challenge page. If the challenge page still appears, the token may be invalid or the IP may be blocked.

#### Implementation Best Practices

1. **Consistent Identity**: Use the same User-Agent, Accept-Language, and header order across all requests. Mismatches between these values will cause detection.
2. **Header Order**: Maintain consistent header ordering between requests. Incapsula checks for header order consistency as part of its fingerprinting.
3. **TLS Fingerprint**: Use a TLS client that matches your User-Agent. For example, if your User-Agent claims Chrome 133, your TLS fingerprint should match Chrome 133.
4. **Token Renewal**: The token has an expiration time (`renewInSec`). Implement a renewal mechanism to avoid re-solving the full challenge.
5. **Public IP**: Your public IP must be obtained through the same proxy/network path you use for all other requests.

#### API Integration Notes

Our API simplifies the complex process of generating valid Reese payloads.

For detailed API specifications, endpoint documentation, and usage examples, please refer to our [API Reference Documentation](https://claude.ai/api-reference/incapsula.md).


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/incapsula/reese84-dynamic.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
