# SBSD Challenge Flow

### Understanding SBSD Protection

State Based Scraping Detection (SBSD) is an advanced bot protection mechanism used by Akamai to protect websites from scrapers. Websites protected by SBSD present a challenge that must be solved before allowing access to the protected content.

### Challenge Flow Overview

The SBSD challenge follows a specific sequence of requests that must be executed in order:

1. **Initial Page Request**: Client makes a request to the protected website
2. **Challenge Page Response**: Server returns a challenge page with a script reference
3. **Script Request**: Client fetches the referenced script
4. **Payload Construction & Submission**: Client generates and submits the required payload
5. **Access Grant**: Upon successful verification, access to the website is granted

### Implementation Guide

#### Step 1: Initial Page Request

Make a standard GET request to the target website. When the site is protected by SBSD, instead of receiving the actual content, you'll receive a challenge page.

<pre class="language-html"><code class="lang-html"><strong>&#x3C;html>
</strong>   &#x3C;body>
      &#x3C;script src="/6mGXhhKgo3Cn/HH/EB0WcpIr3K/X0iuwmY3aY/UkZyWg/Dy/J1CmB4HUQ?v=99b02ce6-f91f-0f49-40ae-6f8493e30211&#x26;t=183446611">&#x3C;/script>
      &#x3C;script>
         (function() {
             var chlgeId = '';
             var scripts = document.getElementsByTagName('script');
             for (var i = 0; i &#x3C; scripts.length; i++) {
                 if (scripts[i].src &#x26;&#x26; scripts[i].src.match(/t=([^&#x26;#]*)/)) {
                     chlgeId = scripts[i].src.match(/t=([^&#x26;#]*)/)[1];
                 }
             }
             var proxied = window.XMLHttpRequest.prototype.send;
             window.XMLHttpRequest.prototype.send = function() {
                 var pointer = this
                 var intervalId = window.setInterval(function() {
                     if (pointer.readyState === 4 &#x26;&#x26; pointer.responseURL &#x26;&#x26; pointer.responseURL.indexOf('t=' + chlgeId) > -1) {
                         location.reload(true);
                         clearInterval(intervalId);
                     }
                 }, 1);
                 return proxied.apply(this, [].slice.call(arguments));
             };
         })();
      &#x3C;/script>
   &#x3C;/body>
&#x3C;/html>
                                    
</code></pre>

#### Step 2: Identify Challenge Signature

The response will contain an HTML page with a script tag that has essential parameters for solving the challenge. You need to extract:

* **Path**: The script URL path
* **v parameter**: A UUID value
* **t parameter**: A challenge token

Example script tag:

```html
<script src="/6mGXhhKgo3Cn/HH/EB0WcpIr3K/X0iuwmY3aY/UkZyWg/Dy/J1CmB4HUQ?v=99b02ce6-f91f-0f49-40ae-6f8493e30214&t=183446612"></script>
```

From this example:

* **Path**: `/6mGXhhKgo3Cn/HH/EB0WcpIr3K/X0iuwmY3aY/UkZyWg/Dy/J1CmB4HUQ`
* **v parameter**: `99b02ce6-f91f-0f49-40ae-6f8493e30214`
* **t parameter**: `183446612`

Implement a regular expression to extract these values:

```javascript
const regex = /([a-z\d/\-_\.]+)\?v=(.*?)(?:&.*?t=(.*?))?["']/i;
const matches = html.match(regex);
const path = matches[1];
const v = matches[2];
const t = matches[3] || "";
```

#### Step 3: Fetch Challenge Script

Request the script using the extracted components:

```http
GET /[path]?v=[v_parameter]&t=[t_parameter] HTTP/2
Headers...
```

You'll need to save this script content for use in the next step.

#### Step 4: Generate and Submit Payload

Using our API service, generate the SBSD payload by providing:

1. The extracted UUID (v parameter)
2. The page URL
3. The script content
4. Your User-Agent
5. Any existing sbsd\_o cookie value, or the bm\_so cookie value if sbsd\_o is not present.

```
POST /sbsd HTTP/1.1
Content-Type: application/json

{
  "userAgent": "Mozilla/5.0...",
  "uuid": "99b02ce6-f91f-0f49-40ae-6f8493e30214",
  "pageUrl": "https://example.com/",
  "o": "existing_sbsd_o_cookie_value_if_any",
  "script": "script_content_from_step_3",
  "ip": "your ipv4 or ipv6 address",
  "acceptLanguage": "en-US,en;q=0.9"
}
```

Our API will return the properly formatted payload string.

#### Step 5: Submit the Solution

POST the generated payload to the challenge endpoint:

```http
POST /[path]?t=[t_parameter] HTTP/2
Headers...

{
  "body": "YOUR_GENERATED_PAYLOAD"
}
```

#### Step 6: Access Protected Content

If the payload is correct, you can now make requests to the protected website and receive the actual content instead of the challenge page.

```http
GET / HTTP/2
Headers...
```

### Implementation Example

Here's a pseudocode example showing the complete flow:

```javascript
// Step 1: Initial request
const initialResponse = fetch("https://example.com/");
const html = await initialResponse.text();

// Step 2: Extract challenge parameters
const regex = /([a-z\d/\-_\.]+)\?v=(.*?)(?:&.*?t=(.*?))?["']/i;
const matches = html.match(regex);
const path = matches[1];
const v = matches[2];
const t = matches[3] || "";

// Step 3: Fetch script
const scriptUrl = `https://example.com${path}?v=${v}&t=${t}`;
const scriptResponse = await fetch(scriptUrl);
const scriptContent = await scriptResponse.text();

// Step 4: Generate payload using our API
const payload = await generatePayload({
  userAgent: "YOUR_USER_AGENT",
  uuid: v,
  pageUrl: "https://example.com/",
  oCookie: getCookie("sbsd_o"),
  script: scriptContent,
  ip: yourIp,
  acceptLanguage: yourAcceptLanguage
});

// Step 5: Submit payload
const submitUrl = `https://example.com${path}?t=${t}`;
await fetch(submitUrl, {
  method: "POST",
  headers: {
    "Content-Type": "application/json"
  },
  body: JSON.stringify({ body: payload })
});

// Step 6: Access protected content
const protectedContent = await fetch("https://example.com/");
```

### Important Notes

1. **Keep Headers Consistent**: Align your headers with Chrome using a TLS client.
2. **Header Order Matters**: Akamai verifies header ordering on all requests
3. **Cookie Management**: Properly store and reuse any cookies set by the server

### Integration with Our API

Our API simplifies the most complex part of this process - generating the correct payload. By providing the necessary parameters to our service, you receive a properly formatted payload ready for submission.

For detailed integration instructions and API endpoints, refer to our [API Reference Documentation](/api-reference/akamai.md).


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/akamai-web/sbsd-challenge-flow.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
