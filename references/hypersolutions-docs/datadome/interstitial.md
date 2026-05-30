# Interstitial

If you already familiar with solving DataDome challenges, you can either install one of our [SDKs](/readme-1.md) for easy integration, or head over to our [API Reference](/api-reference/datadome.md) if you want to handle the implementation yourself.

## Interstitial

This challenge is served using a 403 status code and a response body that looks as follows:

{% code overflow="wrap" %}

```html
<html>
   <head>
      <title>example.com</title>
      <style>#cmsg{animation: A 1.5s;}@keyframes A{0%{opacity:0;}99%{opacity:0;}100%{opacity:1;}}</style>
   </head>
   <body style="margin:0">
      <p id="cmsg">Please enable JS and disable any ad blocker</p>
      <script data-cfasync="false">var dd={'rt':'i','cid':'AHrlqAAAAAMACAOLE2sBBRMATaDxmw==','hsh':'13C44BAB3C9D728ABD66E2A9F0233C','b':1501854,'s':48047,'host':'geo.captcha-delivery.com'}</script><script data-cfasync="false" src="https://ct.captcha-delivery.com/i.js"></script>
   </body>
</html>
```

{% endcode %}

This is almost the same response you would receive with a Slider challenge, however the reference to this URL: `https://ct.captcha-delivery.com/i.js` is unique to interstitial.

### Parsing the HTML

Before we explain how you can manually parse the required values from the HTML posted above, first are shown code snippets of how it can be done easily with our SDKs:

{% tabs %}
{% tab title="Golang" %}

<pre class="language-go"><code class="lang-go">// reader is an `io.Reader` which holds the response body of the HTML.
<strong>
</strong><strong>// datadomeCookie is the cookie value of the cookie with name "datadome",
</strong>// this cookie is set by the 403 block page.

// referer is the URL that served the 403 block page.

deviceLink, err := datadome.ParseInterstitialDeviceCheckLink(reader, datadomeCookie, referer)
if err != nil {
// Handle the error
}
// deviceLink will look like: https://geo.captcha-delivery.com/interstitial/?...
</code></pre>

{% endtab %}

{% tab title="Python" %}
{% code overflow="wrap" %}

```python
from hyper_sdk.datadome import parse_interstitial_device_check_link

device_link = parse_interstitial_device_check_link(html_content, datadome_cookie, referer)
# device_link will look like: https://geo.captcha-delivery.com/interstitial/?...
```

{% endcode %}
{% endtab %}

{% tab title="JS/TS" %}
{% code overflow="wrap" %}

```javascript
import parseInterstitialDeviceCheckUrl from "hyper-sdk-js/datadome/interstitial.js";

const deviceCheckUrl = parseInterstitialDeviceCheckUrl(
    "", // Block page body
    "", // Value of `datadome` cookie
    "" // Referer, e.g. URL you are trying to access
);
if (deviceCheckUrl === null) {
    // deviceCheckUrl will be null if parseInterstitialDeviceCheckUrl failed to parse it.
}
```

{% endcode %}
{% endtab %}
{% endtabs %}

If you would rather parse the response yourself, you will need to extract the following fields from the `dd` object that can be found in the response body: `cid`, `hsh`, `s`, `b`.\
\
You also need to store the URL that received the block, it is used as the `referer`, and the `datadome` cookie that is set on this blocked request.\
\
You can now build the URL as follows:

{% code overflow="wrap" %}

```
https://geo.captcha-delivery.com/interstitial/?initialCid={cid}&hash={hsh}&cid={datadomeCookie}&referer={referer}&s={s}&b={b}&dm=cd
```

{% endcode %}

We will call this URL the deviceLink from now on.

### Fetching the interstitial script

After having parsed the deviceLink in the previous step, you need to make a GET request to it. Make sure that you are sending the same headers and in the same order as your browser. Read and save the response body of this request as we need it to submit to the Hyper Solutions API.

### Fetching the payload from API

Again we have handy SDK functions to help you with these API calls, if you want to implement the API calls yourself, head over to the [API Reference](/api-reference/datadome.md).\
\
There are 3 values that you need for these API calls:

* userAgent: Your browser's User-Agent.
* deviceLink: The link we have parsed in one of the previous sections.
* html: The full response contents of the GET request you made to the deviceLink URL.

You can then generate the payload with the SDK as follows:

{% tabs %}
{% tab title="Golang" %}
{% code overflow="wrap" %}

```go
payload, headers, err := session.GenerateDataDomeInterstitial(ctx, &hyper.DataDomeInterstitialInput{
// Set the required input fields: userAgent, deviceLink, html
})
if err != nil {
// Handle the error
}
// post the payload, use the headers in subsequent requests when required
```

{% endcode %}
{% endtab %}

{% tab title="Python" %}
{% code overflow="wrap" %}

```python
payload = hyper_session.generate_interstitial_payload(hyper_sdk.DataDomeInterstitialInput(
    user_agent=USER_AGENT,
    device_link=device_check_link,
    html=html_content,
    language=language,
    ip=ip
))
# Use the payload.payload to POST to https://geo.captcha-delivery.com/interstitial/
# Use the payload.headers in subsequent requests when required
```

{% endcode %}
{% endtab %}

{% tab title="JS / TS" %}
{% code overflow="wrap" %}

```javascript
// Generate payload
const payload = await generateInterstitialPayload(session, {
    userAgent: "", // Browser user agent to impersonate
    deviceLink: deviceCheckUrl, // deviceCheckUrl
    html: deviceCheckBody,
    language: language,
    ip: ip,
});
// Use the payload.payload to POST to https://geo.captcha-delivery.com/interstitial/
// Use the payload.headers in subsequent requests when required
```

{% endcode %}
{% endtab %}
{% endtabs %}

### Posting payload, solving challenge

The payload returned by the HyperSolutions API in the previous step is an already concatenated Form Data string, you can POST this to the following URL:

```
https://geo.captcha-delivery.com/interstitial/
```

And make sure you are matching the headers in the order that your browser used.\
\
The response of this POST request will be as follows:

{% code overflow="wrap" %}

```json
{
	"cookie": "datadome=cookievalue; Max-Age=31536000; Domain=.example.com; Path=/; Secure; SameSite=Lax",
	"view": "redirect",
	"url": "https://www.example.com/path"
}
```

{% endcode %}

You can parse the JSON response, and update your cookieJar with the `"cookie"` you received from DataDome.\
\
You have now successfully solved DataDome's interstitial challenge, retrying the request that served this block should not give you a challenge anymore.


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/datadome/interstitial.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
