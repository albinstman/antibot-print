# Vercel BotID

#### Overview

Vercel BotID is a bot detection system that can be used alongside Kasada protection. When enabled, protected endpoints require an `x-is-human` header containing a generated token. You can identify sites using Vercel BotID by observing the `x-is-human` header in browser requests to protected endpoints.

#### Step 1: Fetch the c.js Script

Make a GET request to the BotID script. The script path follows this pattern:

{% code overflow="wrap" %}

```
https://www.example.com/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/a-4-a/c.js?i=0&v=3&h=www.example.com
```

{% endcode %}

**Critical requirements:**

* Match browser headers exactly
* Maintain the same header order as Chrome

Save the JavaScript response body as you'll need it for the next step.

#### Step 2: Generate the x-is-human Header via API

Use the Hyper Solutions API to generate the `x-is-human` header value.

Refer to the botid.md and the SDK documentation for accurate fields.

{% tabs %}
{% tab title="Golang" %}
{% code overflow="wrap" %}

```go
header, err := session.GenerateBotIDHeader(ctx, &hyper.BotIDHeaderInput{
    Script:         scriptBody,
    UserAgent:      "your-user-agent",
    IP:             "your-proxy-ip",
    AcceptLanguage: "en-US,en;q=0.9",
})
if err != nil {
    // Handle the error
}
// header is ready to use as x-is-human value
```

{% endcode %}
{% endtab %}

{% tab title="Python" %}
{% code overflow="wrap" %}

```python
from hyper_sdk import BotIDHeaderInput

header = session.generate_botid_header(BotIDHeaderInput(
    script=script_body,
    user_agent="your-user-agent",
    ip="your-proxy-ip",
    accept_language="en-US,en;q=0.9",
))
```

{% endcode %}
{% endtab %}

{% tab title="JS / TS" %}
{% code overflow="wrap" %}

```javascript
import { BotIDHeaderInput, generateBotIDHeader } from 'hyper-sdk-js';

const header = await generateBotIDHeader(session, new BotIDHeaderInput({
    script: scriptBody,
    userAgent: "your-user-agent",
    ip: "your-proxy-ip",
    acceptLanguage: "en-US,en;q=0.9",
}));
```

{% endcode %}
{% endtab %}
{% endtabs %}

#### Step 3: Making Requests to Protected Endpoints

Include the generated token in the `x-is-human` header on requests to protected endpoints. **Observe what headers the browser includes** and match them exactly.

#### When to Re-generate the Header

You may need to generate a new `x-is-human` header if:

* You receive a 429 response on protected endpoints
* Your proxy IP address changes

#### Summary

The complete flow:

1. ✅ GET request to `c.js` script endpoint
2. ✅ Generate `x-is-human` header via Hyper Solutions API
3. ✅ Make requests to protected endpoints with the `x-is-human` header

You have now successfully integrated Vercel BotID!


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/k4sada/vercel-botid.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
