# Reese84

#### Locating the Script Path

The Reese84 script is served from an obscure path unique to each site. You can find it by inspecting network requests in your browser's developer tools and looking for POST requests whose URL contains `?d=` , these are sensor submissions. The corresponding script path is static per site, so you only need to find it once.

#### Implementation Steps

**Step 1: Fetch the Script Content**

Make a GET request to the script URL (the path you located above, including its query parameters). Store the full response body, our API requires the script content to generate a valid sensor.

```http
GET /path/to/reese84/script?s=... HTTP/2
Chrome: Headers
Accept: */*
Sec-Fetch-Dest: script
Sec-Fetch-Mode: no-cors
Sec-Fetch-Site: same-origin
```

**Step 2: Generate & Submit the Sensor**

Use our API to generate a Reese84 sensor, passing the script content along with your other parameters. Then submit the returned sensor to the script path with `?d=yourdomain.com` appended:

```http
POST /path/to/reese84/script?d=www.example.com HTTP/2
Chrome: Headers
Content-Type: text/plain; charset=utf-8

[YOUR_GENERATED_SENSOR]
```

**Step 3: Store the Cookie**

Parse the token from the response and save it as a cookie named `reese84`:

```json
{
  "token": "3:abc123...",
  "renewInSec": 896,
  "cookieDomain": "www.example.com"
}
```

Set the cookie with the domain from the response, then use it in subsequent requests to the protected site.

#### Notes

* The script path is static per site, locate it once and reuse it.
* The token expires after `renewInSec` seconds. Implement renewal before expiration.
* Use consistent headers, User-Agent, and TLS fingerprint across all requests.


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/incapsula/reese84.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
