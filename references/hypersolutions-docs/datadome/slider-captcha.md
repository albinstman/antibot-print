# Slider (captcha)

If you already familiar with solving DataDome challenges, you can either install one of our [SDKs](/readme-1.md) for easy integration, or head over to our [API Reference](/api-reference/datadome.md) if you want to handle the implementation yourself.

## Slider

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
      <script data-cfasync="false">var dd={'rt':'c','cid':'AHrlqAAAAAMA6gifcHcCX3IATaDxmw==','hsh':'EC3A9FB6F2A31D3AF16C270E6531D2','t':'fe','s':43337,'e':'5e8c40553ff322bdcba3d8e59224b6a2858cf1080c5e0f3923cc0fcd3d4217d5','host':'geo.captcha-delivery.com'}</script><script data-cfasync="false" src="https://ct.captcha-delivery.com/c.js"></script>
   </body>
</html>
```

{% endcode %}

{% hint style="warning" %}
If in the response, `t` is set to `bv`, it means your proxy is hard blocked, solving the challenge will not have any effect.
{% endhint %}

This is almost the same response you would receive with a Interstitial challenge, however the reference to this URL: `https://ct.captcha-delivery.com/c.js` is unique to slider.

### Parsing the HTML

Before we explain how you can manually parse the required values from the HTML posted above, first are shown code snippets of how it can be done easily with our SDKs:

{% tabs %}
{% tab title="Golang" %}
{% code overflow="wrap" %}

```go
// reader is an `io.Reader` which holds the response body of the HTML.

// datadomeCookie is the cookie value of the cookie with name "datadome",
// this cookie is set by the 403 block page.

// referer is the URL that served the 403 block page.

deviceLink, err := datadome.ParseSliderDeviceCheckLink(reader, datadomeCookie, referer)
if err != nil {
    // Handle the error
}
// deviceLink will look like: https://geo.captcha-delivery.com/captcha/?...
```

{% endcode %}
{% endtab %}

{% tab title="Python" %}
{% code overflow="wrap" %}

```python
from hyper_sdk.datadome import parse_slider_device_check_link

device_link = parse_slider_device_check_link(html_content, datadome_cookie, referer)
# device_link will look like: https://geo.captcha-delivery.com/captcha/?...
```

{% endcode %}
{% endtab %}

{% tab title="JS / TS" %}
{% code overflow="wrap" %}

```javascript
import {parseSliderDeviceCheckUrl, generateSliderPayload} from "hyper-sdk-js/datadome/slider.js";

const result = parseSliderDeviceCheckUrl(
    "", // Block page body
    "", // Value of `datadome` cookie
    "" // Referer, e.g. URL you are trying to access
);
if (result.isIpBanned) {
    // IP address is banned.
    // Note: result.url is null if this is true.
    return;
}
```

{% endcode %}
{% endtab %}
{% endtabs %}

If you would rather parse the response yourself, you will need to extract the following fields from the `dd` object that can be found in the response body: `cid`, `hsh`, `t`, `s`, `e`.\
\
You also need to store the URL that received the block, it is used as the `referer`, and the `datadome` cookie that is set on this blocked request.\
\
You can now build the URL as follows:

{% code overflow="wrap" %}

```
https://geo.captcha-delivery.com/captcha/?initialCid={cid}&hash={hsh}&cid={datadomeCookie}&t={t}&referer={referer}&s={s}&e={e}&dm=cd
```

{% endcode %}

### Fetching the slider script

After having parsed the deviceLink in the previous step, you need to make a GET request to it. Make sure that you are sending the same headers and in the same order as your browser. Read and save the response body of this request as we need it to submit to the Hyper Solutions API.

### Fetching the slider puzzle

In order for the Hyper Solutions API to solve the slider challenge, it needs access to the images involved. The HTML/JavaScript that we have retrieved in the previous step contains data like this:

{% code overflow="wrap" %}

```javascript
captchaChallengeSeed: '17af5b20aafd238256f5a5d11cf475da',
captchaChallengePath: 'https://dd.prod.captcha-delivery.com/image/2026-01-19/17af5b20aafd238256f5a5d11cf475da.jpg',
```

{% endcode %}

#### Parsing the Image URLs

You need to extract the `captchaChallengePath` value, which gives you the puzzle image URL directly. To get the piece image URL, simply replace `.jpg` with `.frag.png`.

#### Finding the puzzle link (jpg)

You can parse `captchaChallengePath` using this regex:

```regexp
captchaChallengePath:\s*['"]([^'"]+\.jpg)['"]
```

Or if you already have the path extracted:

```regexp
(https:\/\/dd\.prod\.captcha-delivery\.com\/image\/.*?\.jpg)
```

#### Deriving the piece link (frag.png)

Once you have the puzzle URL, derive the piece URL by replacing the extension:

```javascript
const pieceUrl = puzzleUrl.replace('.jpg', '.frag.png');
```

For example:

* **Puzzle URL:** `https://dd.prod.captcha-delivery.com/image/2026-01-19/17af5b20aafd238256f5a5d11cf475da.jpg`
* **Piece URL:** `https://dd.prod.captcha-delivery.com/image/2026-01-19/17af5b20aafd238256f5a5d11cf475da.frag.png`

### Fetching the Images

You need to make a GET request to both URLs and store both responses (base64 encoded):

* The `.jpg` response should be stored as `puzzle`
* The `.frag.png` response should be stored as `piece`

### Fetching the payload from API

Again we have handy SDK functions to help you with these API calls, if you want to implement the API calls yourself, head over to the [API Reference](/api-reference/datadome.md).\
\
There are 5 values that you need for these API calls:

* userAgent: Your browser's User-Agent.
* deviceLink: The link we have parsed in one of the previous sections.
* html: The full response contents of the GET request you made to the deviceLink URL.
* puzzle: The base64 encoded response bytes from the jpg image.
* piece: The base64 encoded response bytes from the `.frag.png` image.

You can then generate the payload with the SDK as follows:

{% tabs %}
{% tab title="Golang" %}
{% code overflow="wrap" %}

```go
checkUrl, headers, err := session.GenerateDataDomeSlider(ctx, &hyper.DataDomeSliderInput{
    // Set the required input fields
})
if err != nil {
    // Handle the error
}
// Create a GET request to the checkUrl
// Use the headers in subsequent requests when required
```

{% endcode %}
{% endtab %}

{% tab title="Python" %}
{% code overflow="wrap" %}

```python
payload = hyper_session.generate_slider_payload(hyper_sdk.DataDomeSliderInput(
    # fields here
))
# Create a GET request to the payload.payload
# Use the extra headers included with payload.headers when applicable
```

{% endcode %}
{% endtab %}

{% tab title="JS / TS" %}
{% code overflow="wrap" %}

```javascript
// Response body from doing a GET request to result.url
const deviceCheckBody = "";

const payload = await generateSliderPayload(session, {
    // fields here
});
// Create a GET request to the payload.payload
// Use the extra headers included with payload.headers when applicable
```

{% endcode %}
{% endtab %}
{% endtabs %}

### Posting payload, solving challenge

Our API returns a simple URL as the result of solving the slider challenge, all that is required is to make a GET request to this URL which looks as follows:

```
https://geo.captcha-delivery.com/captcha/check?cid=...
```

The response of this GET request will be as follows:

{% code overflow="wrap" %}

```json
{
	"cookie": "datadome=cookievalue; Max-Age=31536000; Domain=.example.com; Path=/; Secure; SameSite=Lax"
}
```

{% endcode %}

You can parse the JSON response, and update your cookieJar with the `"cookie"` you received from DataDome.\
\
You have now successfully solved DataDome's slider challenge, retrying the request that served this block should not give you a challenge anymore.


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/datadome/slider-captcha.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
