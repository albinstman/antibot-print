# Getting started

## Sensor Data

If you're already familiar with Akamai Bot Manager challenges, you can either install one of our [SDKs](/readme-1.md) for easy integration, or head over to our [API Reference](/api-reference/akamai.md) if you want to handle the implementation yourself.

### Understanding Akamai Protection

Akamai Bot Manager protects websites by requiring clients to generate and submit sensor data that proves they're legitimate browsers. This protection manifests as:

* A dynamically generated script endpoint embedded in protected pages
* An `_abck` cookie that gets validated when performing protected actions
* Cookie validation that occurs when accessing protected endpoints (login, add to cart, checkout, etc.)

The `_abck` cookie becomes valid after successfully posting sensor data. A cookie containing `~0~` indicates you can stop posting additional sensors, though not all sites use this indicator.

### Solution Flow

#### Step 1: Initial Page Request

Make a GET request to the protected page. This is typically the page users would naturally visit before performing the protected action (e.g., product page before add-to-cart).

**Critical:** You must use a TLS client that mimics Chrome and match the exact header order of real browsers.

#### Step 2: Parse Script Endpoint

Extract the Akamai script endpoint from the HTML response. The script tag is typically located near the end of the body and contains a dynamically generated path:

```html
<script type="text/javascript" src="/yMOlMy/yS/3T/NVx6/a7xTRI1O5hJJ8/EDi7z45Ou1bfXb/dzldXmhnIQk/CjdBHQkD/Hn0" defer></script>
```

**Important:** This path is unique and dynamic - it cannot be hardcoded and must be parsed from each response.

{% tabs %}
{% tab title="Golang" %}

```go
import "github.com/Hyper-Solutions/hyper-sdk-go/v2/akamai"

// Parse script path from HTML reader
scriptPath, err := akamai.ParseScriptPath(htmlReader)
if err != nil {
    // Handle parsing error
}
// scriptPath will be like: /yMOlMy/yS/3T/NVx6/a7xTRI1O5hJJ8/...
```

{% endtab %}

{% tab title="Python" %}

```python
from hyper_sdk.akamai import parse_script_path

script_path = parse_script_path(html_content)
# script_path will be like: /yMOlMy/yS/3T/NVx6/a7xTRI1O5hJJ8/...
```

{% endtab %}

{% tab title="JS / TS" %}

```javascript
import { parseAkamaiPath } from "hyper-sdk-js";

const scriptPath = parseAkamaiPath(htmlContent);
// scriptPath will be like: /yMOlMy/yS/3T/NVx6/a7xTRI1O5hJJ8/...
```

{% endtab %}
{% endtabs %}

#### Step 3: Fetch Script Content

Request the script content from the parsed endpoint. Save the entire response body as you'll need it for sensor generation.

Remember to:

* Use the same TLS client
* Include appropriate referer
* Maintain consistent cookie jar

#### Step 4: Generate Sensor Data

Use the Hyper Solutions API to generate sensor data. The sensor data simulates complex browser behavior and environment fingerprinting:

{% tabs %}
{% tab title="Golang" %}

```go
sensorData, sensorContext, err := session.GenerateSensorData(ctx, &hyper.SensorInput{
    PageUrl:        "https://www.example.com/product/example-item",
    UserAgent:      userAgent,
    Abck:           currentAbckCookie,  // Current _abck cookie value
    Bmsz:           bmSzCookie,         // bm_sz cookie value
    Version:        "3",                // Akamai version (usually "3")
    Script:         scriptContent,      // Full script content (first request only)
    Context:        sensorContext,      // Previous context (empty on first request)
    AcceptLanguage: "en-US,en;q=0.9",
    IP:             clientIP,           // Required: client IP address
})
if err != nil {
    // Handle error
}
```

{% endtab %}

{% tab title="Python" %}

```python
from hyper_sdk import SensorInput

sensor_data, sensor_context = session.generate_sensor_data(SensorInput(
    page_url="https://www.example.com/product/example-item",
    user_agent=user_agent,
    abck=current_abck_cookie,  # Current _abck cookie value
    bmsz=bm_sz_cookie,         # bm_sz cookie value
    version="3",               # Akamai version
    script=script_content,     # Full script content (first request only)
    context=sensor_context,    # Previous context (empty on first request)
    accept_language="en-US,en;q=0.9",
    ip=client_ip              # Required: client IP address
))
```

{% endtab %}

{% tab title="JS / TS" %}

```javascript
import { SensorInput, generateSensorData } from "hyper-sdk-js";

const result = await generateSensorData(session, new SensorInput(
    "https://www.example.com/product/example-item",  // pageUrl
    userAgent,
    currentAbckCookie,    // Current _abck cookie value
    bmSzCookie,          // bm_sz cookie value
    "3",                 // Akamai version
    scriptContent,       // Full script content (first request only)
    sensorContext,       // Previous context (empty on first request)
    "en-US,en;q=0.9",   // acceptLanguage
    clientIP            // Required: client IP address
));

const sensorData = result.payload;
const newSensorContext = result.context;
```

{% endtab %}
{% endtabs %}

**Important notes about sensor generation:**

* The `script` parameter is only needed on the first sensor generation
* The `context` parameter should be empty on first request, then use the returned context for subsequent requests
* Save the returned `sensorContext` for use in the next sensor generation

#### Step 5: Submit Sensor Data

POST the generated sensor data to the same script endpoint. The payload should be JSON formatted with a single `sensor_data` field:

```json
{"sensor_data":"[generated_sensor_data_string]"}
```

The response will update your `_abck` cookie through Set-Cookie headers.

#### Step 6: Validate and Repeat

Check if the updated `_abck` cookie indicates completion:

{% tabs %}
{% tab title="Golang" %}

```go
// Check for the ~0~ pattern (when available)
if strings.Contains(abckCookieValue, "~0~") {
    // Can stop posting sensors
}

// Or use the validation helper
isValid := akamai.IsCookieValid(abckCookieValue, requestCount)
```

{% endtab %}

{% tab title="Python" %}

```python
# Check for the ~0~ pattern (when available)
if "~0~" in abck_cookie_value:
    # Can stop posting sensors

# Or use the validation helper
is_valid = is_cookie_valid(abck_cookie_value, request_count)
```

{% endtab %}

{% tab title="JS / TS" %}

```javascript
// Check for the ~0~ pattern (when available)
if (abckCookieValue.includes("~0~")) {
    // Can stop posting sensors
}

// Or use the validation helper
const isValid = isAkamaiCookieValid(abckCookieValue, requestCount);
```

{% endtab %}
{% endtabs %}

**Sensor posting strategy:**

* If the cookie contains `~0~`, you can proceed to the protected action
* If the site doesn't use the `~0~` indicator, post exactly 3 sensors before proceeding
* Each subsequent sensor should NOT include the script content (only needed on first request)
* Each subsequent sensor MUST use the context returned from the previous generation

#### Step 7: Perform Protected Action

Once you have a valid `_abck` cookie (either containing `~0~` or after posting 3 sensors), you can proceed with the protected action.

**Important:** After performing a protected action, the `_abck` cookie typically becomes invalidated. You might need to generate new sensor data before the next protected action.

### Critical Implementation Requirements

#### TLS Client Configuration

**You MUST use a TLS client that:**

* Supports modern TLS cipher suites
* Can maintain exact header order
* Properly handles HTTP/2 or HTTP/1.1 as the target site requires
* Maintains consistent TLS fingerprint throughout the session

Using standard HTTP clients without proper TLS configuration will result in detection and blocking.

#### Header Order

**Header order is critical.** Akamai's detection system analyzes the exact order of HTTP headers. You must:

* Match the header order of real browsers
* Maintain consistent header order throughout all requests
* Use a client that allows precise header order control

#### Session Consistency

Throughout the entire flow, maintain:

* **Same User-Agent** for all requests
* **Same TLS fingerprint** across all connections
* **Proper cookie forwarding** between requests
* **Consistent client IP address** for all operations

### Best Practices

1. **Parse Dynamic Paths**: Never hardcode script endpoints - they change regularly and are unique per session
2. **Context Preservation**: Always save and reuse the sensor context between generations
3. **Script Caching**: The script content only needs to be fetched once per session (use it only for the first sensor)
4. **Retry Limits**: Post a maximum of 3 sensors - if unsuccessful, the issue is likely with your TLS client or header configuration
5. **Cookie Monitoring**: Check for cookie invalidation after each protected action
6. **IP Consistency**: Use the same IP address throughout the entire session

### Troubleshooting

#### Sensors Not Generating Valid Cookies

* Verify your TLS client configuration matches browser fingerprints
* Ensure header order exactly matches browser patterns
* Confirm the IP address is consistent and not blacklisted
* Check that you're properly parsing the dynamic script path

#### Immediate Detection/Blocking

* Your TLS fingerprint is likely incorrect or inconsistent
* Header order doesn't match expected browser patterns
* The client IP may be from a datacenter or known proxy range

#### Cookie Invalidates Too Quickly

* This is normal after protected actions - regenerate before each protected request
* Some sites invalidate after a specific number of requests regardless of actions
* Ensure you're not reusing invalidated cookies

For more detailed API documentation, refer to our [API Reference](/api-reference/authentication.md) or check our [SDKs](/readme-1.md) for your preferred programming language.

### Complete Example

For a full working implementation with proper TLS client setup, header ordering, and cookie handling, see our examples repository:

{% embed url="<https://github.com/Hyper-Solutions/hypersolutions-examples>" %}


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/akamai-web/getting-started.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
