# SBSD Introduction

#### Understanding Basic SBSD Protection

State Based Scraping Detection (SBSD) is an advanced bot protection mechanism used by Akamai. While our other guides cover handling SBSD challenges and 429 blocks, many websites implement SBSD in a passive mode that simply requires posting sensors proactively to maintain access.

#### Identifying Basic SBSD

When you request a page with basic SBSD protection, you'll receive normal page content along with an SBSD script reference:

```html
<script src="/6mGXhhKgo3Cn/HH/EB0WcpIr3K/X0iuwmY3aY/UkZyWg/Dy/J1CmB4HUQ?v=99b02ce6-f91f-0f49-40ae-6f8493e30214"></script>
```

Note the absence of the `t` parameter - this distinguishes basic SBSD from hard challenges. The script only contains:

* **Path**: The script URL path
* **v parameter**: A UUID value
* **No t parameter**: This only appears in challenge scenarios

#### Implementation Flow

Basic SBSD protection follows a simple sequence:

1. **Initial Page Request**: Client requests the protected page
2. **Parameter Extraction**: Extract path and UUID from the script tag
3. **Script Fetch**: Request the SBSD script
4. **Sensor Submission**: Post two SBSD sensors (index 0 and 1)
5. **Continue Normally**: Proceed with your intended requests

#### Implementation Guide

**Step 1: Extract SBSD Parameters**

Parse the HTML response to extract the SBSD script parameters:

```javascript
const regex = /([a-z\d/\-_\.]+)\?v=([^"'&]+)/i;
const matches = html.match(regex);
const path = matches[1];
const uuid = matches[2];
```

**Step 2: Fetch the SBSD Script**

Request the script using the extracted parameters:

```http
GET /[path]?v=[uuid] HTTP/2
Headers...
```

Save the script content for use in sensor generation.

**Step 3: Generate and Submit First Sensor**

Using our API service, generate the first SBSD payload with index 0:

```json
POST /sbsd HTTP/1.1
Content-Type: application/json

{
  "userAgent": "Mozilla/5.0...",
  "uuid": "99b02ce6-f91f-0f49-40ae-6f8493e30214",
  "pageUrl": "https://example.com/",
  "o": "cookie_value",
  "script": "script_content_from_step_2",
  "ip": "your_ip_address",
  "acceptLanguage": "en-US,en;q=0.9",
  "index": 0
}
```

Submit the generated payload to the SBSD endpoint:

```http
POST /[path] HTTP/2
Content-Type: application/json

{
  "body": "GENERATED_PAYLOAD_INDEX_0"
}
```

**Step 4: Generate and Submit Second Sensor**

Immediately follow with the second sensor using index 1:

```json
POST /sbsd HTTP/1.1
Content-Type: application/json

{
  "userAgent": "Mozilla/5.0...",
  "uuid": "99b02ce6-f91f-0f49-40ae-6f8493e30214",
  "pageUrl": "https://example.com/",
  "o": "cookie_value",
  "script": "script_content_from_step_2",
  "ip": "your_ip_address",
  "acceptLanguage": "en-US,en;q=0.9",
  "index": 1
}
```

Submit to the same endpoint:

```http
POST /[path] HTTP/2
Content-Type: application/json

{
  "body": "GENERATED_PAYLOAD_INDEX_1"
}
```

**Step 5: Proceed with Protected Requests**

After posting both sensors, you can make requests to protected endpoints without triggering SBSD challenges.

#### Implementation Example

Here's a complete flow in pseudocode:

```javascript
// Step 1: Initial request
const response = await fetch("https://example.com/");
const html = await response.text();

// Step 2: Extract SBSD parameters
const regex = /([a-z\d/\-_\.]+)\?v=([^"'&]+)/i;
const matches = html.match(regex);

if (!matches) {
  // No SBSD protection, continue normally
  return;
}

const path = matches[1];
const uuid = matches[2];

// Step 3: Fetch SBSD script
const scriptUrl = `https://example.com${path}?v=${uuid}`;
const scriptResponse = await fetch(scriptUrl);
const scriptContent = await scriptResponse.text();

// Step 4: Post both sensors
const postUrl = `https://example.com${path}`;

for (let index = 0; index < 2; index++) {
  // Generate payload using our API
  const payload = await generatePayload({
    userAgent: "YOUR_USER_AGENT",
    uuid: uuid,
    pageUrl: "https://example.com/",
    oCookie: getCookie("bm_so") || getCookie("sbsd_o"),
    script: scriptContent,
    ip: yourIp,
    acceptLanguage: "en-US,en;q=0.9",
    index: index
  });
  
  // Submit sensor
  await fetch(postUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ body: payload })
  });
}

// Step 5: Continue with protected requests
const apiResponse = await fetch("https://example.com/api/data");
```

#### Key Differences from Challenge-Based SBSD

Basic SBSD protection differs from challenge scenarios:

1. **No blocking page** - Normal content loads immediately
2. **No t parameter** - Script URL only contains the v parameter
3. **Two sensors required** - Always post index 0 and index 1
4. **Proactive protection** - Sensors prevent future blocks rather than solving existing ones

#### Important Notes

1. **Cookie Management**: Use the `bm_so` or `sbsd_o` cookie value for the `o` parameter.
2. **Script Reuse**: The SBSD script content can be cached and reused for multiple sensor posts within the same session.
3. **Header Consistency**: Maintain consistent headers across all requests, including User-Agent and Accept-Language.
4. **Sensor Order**: Always post sensors in order (index 0 first, then index 1).

For handling blocking challenges or 429 responses with challenge tokens, refer to:

* [SBSD Challenge Flow](/akamai-web/sbsd-challenge-flow.md)
* [Handling 429 Status Codes](/akamai-web/handling-429-status-codes-with-sbsd-challenges.md)


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/akamai-web/sbsd-introduction.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
