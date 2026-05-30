# Tags

Unlike slider and interstitial, tags will never be served in a challenge or block page. This type is used to send extra telemetry data to DataDome that will increase trust score of the session (resulting in less blocks). \
\
In browser you will see multiple requests to a `/js`endpoint, this is where browser sends the tags data. This is how you can generate this data using our SDKS:

{% tabs %}
{% tab title="Golang" %}

```go
payload, err := session.GenerateDataDomeTags(ctx, &hyper.DataDomeTagsInput{
    UserAgent: "", // Your chrome useragent
    Cid: "", // Your current datadome cookie
    Ddk: "", // sitekey, static for each site. parse it from the /js/ payload request from browser
    Referer: "", // The referer visible as the referer header in the payload POST
    Type: "", // First time 'ch', second time 'le'
    Language: "", // The first language of your accept-language header, defaults to "en-US"
    IP: "", // The IP that is used to post the sensor data to the target site. You can use /ip to get the IP from a connection. If you are not using proxies, this will be the IPv4 address of your pc.
})
if err != nil {
// Handle the error
}
// Use the payload to POST to /js
```

{% endtab %}

{% tab title="Python" %}

```python
payload = hyper_session.generate_tags_payload(hyper_sdk.DataDomeTagsInput(
    user_agent=USER_AGENT, # Your chrome UserAgent
    cid=cid, # Your current datadome cookie
    ddk=ddk, # sitekey, static for each site. parse it from the /js/ payload request from browser
    referer=referer, # The referer visible as the referer header in the payload POST
    tags_type=tags_type, # First time 'ch', second time 'le'
    ip=ip, # The IP that is used to post the sensor data to the target site. You can use /ip to get the IP from a connection. If you are not using proxies, this will be the IPv4 address of your pc.
    accept_language=accept_language# The accept language header that you use
))
# Use the payload to POST to /js
```

{% endtab %}

{% tab title="JS / TS" %}

```javascript
const payload = await generateTagsPayload(session, {
    userAgent: "", // Your chrome UserAgent
    cid: cid, // Your current datadome cookie
    ddk: ddk, // sitekey, static for each site. parse it from the /js/ payload request from browser
    referer: referer, // The referer visible as the referer header in the payload POST
    type: type, // First time 'ch', second time 'le'
    ip: ip, // The IP that is used to post the sensor data to the target site. You can use /ip to get the IP from a connection. If you are not using proxies, this will be the IPv4 address of your pc.
    acceptLanguage: language, // The accept-language header value
});
```

{% endtab %}
{% endtabs %}

You need to POST this payload to the `/js`endpoint same way browser will do it. The endpoint will return a response like this:

```json
{
	"status": 200,
	"cookie": "datadome=L7HH_UaWyA17TZFa7FNKxtIE9cReX~6bpf~E5A5IetWsibg0KwHgedPMPHee40cm4VqY9r3Yr6ZOCuWL17WB71PDE92lXdBIyyl3M2SZyhOl~7rmkK_XxE0O19hB4q0o; Max-Age=31536000; Domain=.vinted.fr; Path=/; Secure; SameSite=Lax"
}
```

You should update your `datadome`cookie in your cookiejar manually, with the value returned in the response.\
\
Posting tags  should be done twice, always retrieving the `datadome`cookie from the first tags POST request. First with type `ch`and the second time with type `le`.


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/datadome/tags.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
