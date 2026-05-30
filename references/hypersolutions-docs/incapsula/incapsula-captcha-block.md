# Incapsula Captcha Block

### Challenge Flow Overview

The Incapsula captcha challenge follows this sequence:

1. **Initial Request**: Client requests a protected page
2. **Captcha Block Page**: Server returns a captcha challenge page
3. **Resource Request**: Client requests the embedded resource URL
4. **Token Submission**: Client submits the captcha token to the challenge endpoint
5. **Access Granted**: Upon verification, client can access the protected content

### Implementation Guide

#### Step 1: Initial Request & Challenge Detection

When you make a request to a protected resource, you'll receive a captcha challenge page instead of the expected content:

```http
GET / HTTP/2
Chrome: Headers
```

The response will contain HTML:

```html
<html style="height:100%">
<head>
  <META NAME="ROBOTS" CONTENT="NOINDEX, NOFOLLOW">
  <meta name="format-detection" content="telephone=no">
  <meta name="viewport" content="initial-scale=1.0">
  <meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1">
  <script type="text/javascript" src="/_Incapsula_Resource?SWJIYLWA=719d34d31c8e3a6e6fffd425f7e032f3"></script>
  <script src="/nions-to-vnse-the-Bewarfish-so-like-here-hoa-Mon" async></script>
</head>
<body style="margin:0px;height:100%">
  <iframe id="main-iframe" src="/_Incapsula_Resource?SWUDNSAI=31&xinfo=51-29384756-0%20NNNY%20RT%281744568237689%2043%29%20q%280%20-1%20-1%200%29%20r%280%20-1%29%20B12%2814%2c0%2c0%29%20U18&incident_id=1687000240661649450-106846605907789363&edet=12&cinfo=0e0000000e25&rpinfo=0&cts=XlV9lAHtmM6iAKQa1hQ6Yvp2jH9v9NdRImAO%2fBAIXo7Yl3vQMpnD%2bTzrDt%2f%2bPS32&mth=GET" frameborder=0 width="100%" height="100%" marginheight="0px" marginwidth="0px">
    Request unsuccessful. Incapsula incident ID: ...
  </iframe>
</body>
</html>
```

#### Step 2: Extract Resource URL

Extract the iframe source URL from the challenge page to build this URL:

```
https://www.example.com/_Incapsula_Resource?SWUDNSAI=31&xinfo=51-19749804-0%20NNNY%20RT%281744182960404%2043%29%20q%280%20-1%20-1%200%29%20r%280%20-1%29%20B12%2814%2c0%2c0%29%20U18&incident_id=1687000240661649450-106846605907789363&edet=12&cinfo=0e0000000e25&rpinfo=0&cts=XlV9lAHtmM6iAKQa1hQ6Yvp2jH9v9NdRImAO%2fBAIXo7Yl3vQMpnD%2bTzrDt%2f%2bPS32&mth=GET
```

#### Step 3: Request the Resource

Make a GET request to the extracted resource URL:

```http
GET /_Incapsula_Resource?SWUDNSAI=31&xinfo=51-29384756-0%20NNNY%20RT%281744568237689%2043%29%20q%280%20-1%20-1%200%29%20r%280%20-1%29%20B12%2814%2c0%2c0%29%20U18&incident_id=1687000240661649450-106846605907789363&edet=12&cinfo=0e0000000e25&rpinfo=0&cts=XlV9lAHtmM6iAKQa1hQ6Yvp2jH9v9NdRImAO%2fBAIXo7Yl3vQMpnD%2bTzrDt%2f%2bPS32&mth=GET HTTP/2
Chrome: Headers
```

The response will be an HTML page containing the captcha challenge and most importantly, the POST URL for submitting the token. Parse this response to extract the POST URL:

```javascript
xhr.open("POST", "/_Incapsula_Resource?SWCGHOEL=v2&dai=106846605907789363&cts=XlV9lAHtmM6iAKQa1hQ6Yvp2jH9v9NdRImAO%2fBAIXo7Yl3vQMpnD%2bTzrDt%2f%2bPS32", true);
```

#### Step 4: Obtain Captcha Token

The page contains a hCaptcha or Geetest challenge. You need to solve this captcha and obtain a token.

#### Step 5: Submit the Captcha Token

Post the captcha token to the extracted POST URL:

```http
POST /_Incapsula_Resource?SWCGHOEL=v2&dai=106846605907789363&cts=XlV9lAHtmM6iAKQa1hQ6Yvp2jH9v9NdRImAO%2fBAIXo7Yl3vQMpnD%2bTzrDt%2f%2bPS32 HTTP/2
Chrome: Headers

g-recaptcha-response=tokenhere
```

The server will respond with a Set-Cookie header containing an `incap_sh_*` cookie:

```
HTTP/2 200 OK
Date: Wed, 09 Apr 2025 12:34:56 GMT
Content-Type: text/html; charset=utf-8
Set-Cookie	incap_sh_1979199=sh72ZwAAAAA6PmkpBgAIsr3YvwY8ht+b3wLU6h2+Aq+WkW+w; HttpOnly; Path=/; SameSite=None; Secure; Max-Age=3600
```

#### Step 6: Access the Protected Content

With the `incap_sh_*` cookie set, make your original request again to access the previously protected resource:

```http
GET / HTTP/2
Chrome: Headers
Cookie: incap_sh_1979199=sh72ZwAAAAA6PmkpBgAIsr3YvwY8ht+b3wLU6h2+Aq+WkW+w
```

The server should now return the protected content instead of the captcha challenge.


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/incapsula/incapsula-captcha-block.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
