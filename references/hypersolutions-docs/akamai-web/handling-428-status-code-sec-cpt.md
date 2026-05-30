# Handling 428 Status Code (SEC-CPT)

### Understanding SEC-CPT Blocks

When interacting with APIs protected by Akamai, you may encounter a `428 Precondition Required` status code. This indicates that you have triggered a challenge that must be solved before you can continue making requests.

The challenge response contains a JSON payload with provider-specific information:

```json
{
  "sec-cp-challenge": "true",
  "provider": "crypto",
  ...
}
```

The `provider` field determines which challenge flow you need to follow. The three providers are:

* **crypto** - A proof-of-work challenge with a mandatory wait duration
* **behavioral** - A behavioral analysis challenge requiring normal sensor data
* **adaptive** - A combined challenge requiring both proof-of-work and sensor data submission

#### Key Cookie

The `sec_cpt` cookie is the primary indicator of challenge status. A successfully solved challenge will result in a `sec_cpt` cookie containing `~3~` in its value.

***

### Crypto Provider

The crypto provider implements a proof-of-work challenge with a **mandatory wait duration that cannot be bypassed**.

#### Challenge Response Structure

When you receive a crypto challenge, the response contains:

| Field                  | Description                                                                    |
| ---------------------- | ------------------------------------------------------------------------------ |
| `sec-cp-challenge`     | Always `"true"` indicating an active challenge                                 |
| `provider`             | `"crypto"` for this challenge type                                             |
| `branding_url_content` | Path to the challenge page (e.g., `/_sec/cp_challenge/crypto_message-4-3.htm`) |
| `chlg_duration`        | **Mandatory wait time in seconds**                                             |
| `token`                | Challenge token for payload generation                                         |
| `timestamp`            | Server timestamp                                                               |
| `nonce`                | Cryptographic nonce                                                            |
| `difficulty`           | Proof-of-work difficulty parameter                                             |
| `timeout`              | Challenge timeout value                                                        |

#### Solution Flow

**Step 1: Parse the Challenge**

Extract the challenge data from the 428 response. The response can come in two formats:

* **HTML format**: Contains an iframe with `challenge` attribute (base64-encoded JSON), `data-duration` attribute, and `src` attribute for the challenge path
* **JSON format**: Direct JSON response with all challenge parameters

**Step 2: Wait the Required Duration**

You **must** wait for the duration specified in `chlg_duration` (or `data-duration` in HTML format). This wait time is enforced server-side and cannot be bypassed or shortened.

**Step 3: Generate and Submit the Proof-of-Work Payload**

After waiting, generate the proof-of-work payload containing:

* The challenge `token`
* Computed `answers` based on the challenge parameters (nonce, timestamp, difficulty)

Submit this payload via POST to `/_sec/verify?provider=crypto` on the target domain.

**Step 4: Verify the Challenge**

Make a GET request to `/_sec/cp_challenge/verify` to complete the verification process.

**Step 5: Validate Success**

Check that the `sec_cpt` cookie now contains `~3~` in its value. If not, the challenge was not successfully solved.

***

### Behavioral Provider

The behavioral provider requires sensor data submission, similar to standard Akamai sensor flow but with a different endpoint structure.

#### Challenge Response Structure

```json
{
  "sec-cp-challenge": "true",
  "provider": "behavioral",
  "branding_type": "custom_branding",
  "branding_cust_url": "/challenge.html",
  "verify_url": "fwrjQWEM/6OcbaAS/TQkjRzC/-A/wXa1S2iYkY/IBwoXw/AD5PIWQF/GUEB"
}
```

| Field               | Description                                              |
| ------------------- | -------------------------------------------------------- |
| `sec-cp-challenge`  | Always `"true"` indicating an active challenge           |
| `provider`          | `"behavioral"` for this challenge type                   |
| `branding_type`     | Branding configuration type                              |
| `branding_cust_url` | Path to the challenge branding page                      |
| `verify_url`        | **Dynamic verification URL path** (unique per challenge) |

#### Solution Flow

**Step 1: Fetch the Branding Page**

Make a GET request to the `branding_cust_url` path (e.g., `/challenge.html`) on the target domain. This page contains the script endpoint needed for sensor submission.

**Step 2: Extract and Fetch the Script**

Parse the branding page response to locate the Akamai script endpoint. Make a GET request to fetch the script content - this is required for sensor generation.

**Step 3: Submit Sensor Data**

Generate and POST sensor data to the script endpoint. Key considerations:

* Use a **fresh sensor context** for this challenge (do not reuse context from previous requests)
* While one sensor POST is often sufficient, implement a loop of up to **3 sensor posts**
* Break out of the loop early if the `sec_cpt` cookie is set (indicates sufficient sensor data was received)
* Include the `_abck` cookie value in sensor generation

**Step 4: Verify the Challenge**

Make a GET request to the `verify_url` path returned in the original challenge response. Note that this is a **dynamic path** unique to each challenge, not the static `/_sec/cp_challenge/verify` endpoint used by the crypto provider.

**Step 5: Validate Success**

Confirm that the `sec_cpt` cookie contains `~3~` in its value.

***

### Adaptive Provider

The adaptive provider combines both proof-of-work and sensor data submission into a single sequential flow. It effectively merges the crypto and behavioral challenge types — you must first complete the proof-of-work step, then submit sensor data, and finally verify.

#### Challenge Response Structure

```json
{
  "sec-cp-challenge": "true",
  "provider": "adaptive",
  "chlg_duration": 30,
  "branding_type": "custom_branding",
  "branding_cust_url": "/challenge-assets/v7/captcha.html",
  "token": "<challenge_token>",
  "timestamp": 1772786267,
  "nonce": "0f7c9e91cbd8ab5f6008",
  "difficulty": 15000,
  "count": 1,
  "timeout": 1000,
  "verify_url": "<dynamic_url>"
}
```

| Field               | Description                                                                  |
| ------------------- | ---------------------------------------------------------------------------- |
| `sec-cp-challenge`  | Always `"true"` indicating an active challenge                               |
| `provider`          | `"adaptive"` for this challenge type                                         |
| `chlg_duration`     | **Mandatory wait time in seconds**                                           |
| `branding_type`     | Branding configuration type                                                  |
| `branding_cust_url` | Path to the challenge branding page                                          |
| `token`             | Challenge token for payload generation                                       |
| `timestamp`         | Server timestamp                                                             |
| `nonce`             | Cryptographic nonce                                                          |
| `difficulty`        | Proof-of-work difficulty parameter                                           |
| `count`             | Number of proof-of-work answers required                                     |
| `timeout`           | Challenge timeout value                                                      |
| `verify_url`        | Present in response but **not used** — verification uses the static endpoint |

#### Solution Flow

**Step 1: Parse the Challenge**

Extract the challenge data from the 428 response. Note that adaptive challenges include fields from both crypto (token, nonce, difficulty, count) and behavioral (branding\_cust\_url) providers.

**Step 2: Wait the Required Duration**

You **must** wait for the duration specified in `chlg_duration`. As with the crypto provider, this wait time is enforced server-side and cannot be bypassed or shortened.

**Step 3: Generate and Submit the Proof-of-Work Payload**

After waiting, generate the proof-of-work payload containing:

* The challenge `token`
* Computed `answers` based on the challenge parameters (nonce, timestamp, difficulty)
* The number of answers must match the `count` field (e.g., if `count` is 1, provide one answer)

Submit this payload via POST to `/_sec/verify?provider=adaptive` on the target domain.

Example payload:

```json
{
  "token": "<challenge_token>",
  "answers": ["0.66463d05840cd"]
}
```

**Step 4: Submit Sensor Data**

After completing the proof-of-work step, proceed with the sensor data submission flow:

1. Fetch the branding page at the `branding_cust_url` path
2. Extract and fetch the Akamai script endpoint from the branding page
3. Generate and POST sensor data to the script endpoint in a loop of up to **3 sensor posts**
4. Break out of the loop early if the `sec_cpt` cookie is set

This follows the same sensor submission process as the behavioral provider — use a fresh sensor context and include the `_abck` cookie value in sensor generation.

**Step 5: Verify the Challenge**

Make a GET request to `/_sec/cp_challenge/verify` (the static verification endpoint). Note that unlike the behavioral provider, the adaptive provider uses the **static** verify endpoint rather than the dynamic `verify_url` from the challenge response.

A successful response will contain:

```json
{"success": "true"}
```

**Step 6: Validate Success**

Confirm that the `sec_cpt` cookie contains `~3~` in its value.

***

### Implementation Best Practices

#### Header Ordering

Maintaining correct header order is critical for all challenge types. Akamai's protection systems analyze header ordering as part of their fingerprinting. Always ensure your HTTP client preserves the exact header order you specify.

#### Cookie Management

* **`sec_cpt`**: Primary challenge status cookie - monitor for `~3~` to confirm success
* **`bm_sz`**: Akamai bot manager cookie - required for sensor generation
* **`_abck`**: Akamai bot manager cookie - required for sensor generation

Properly store and forward cookies between all requests to maintain session state.

#### Session Consistency

* Use the same User-Agent throughout the entire challenge flow
* Maintain consistent client hints (`sec-ch-ua`, `sec-ch-ua-mobile`, `sec-ch-ua-platform`)
* Keep TLS fingerprint consistent across all requests

#### Handling Challenges in API Flows

SEC-CPT challenges can appear in two contexts:

1. **Initial page load**: The challenge appears when first accessing a protected page
2. **During API calls**: A previously valid session may receive a 428 response, requiring challenge resolution before retrying the original request

Implement automatic challenge detection and solving in your API client to handle both scenarios seamlessly.

#### Error Handling

* If the `sec_cpt` cookie does not contain `~3~` after completing the flow, the challenge was not successfully solved
* For behavioral challenges, if sensor posts don't result in a `sec_cpt` cookie after 3 attempts, the implementation is most likely incorrect
* For adaptive challenges, ensure both the proof-of-work submission and sensor posts complete successfully, failure in either phase will prevent verification
* For crypto challenges, ensure you wait the **full duration** before submitting - premature submission will fail

***

### Quick Reference

| Aspect            | Crypto Provider                      | Behavioral Provider                 | Adaptive Provider                                  |
| ----------------- | ------------------------------------ | ----------------------------------- | -------------------------------------------------- |
| Wait Required     | Yes (mandatory)                      | No                                  | Yes (mandatory)                                    |
| Proof-of-Work     | Yes                                  | No                                  | Yes                                                |
| Sensor Posts      | No                                   | Yes (1-3 posts)                     | Yes (1-3 posts, after proof-of-work)               |
| POST Endpoint     | `/_sec/verify?provider=crypto`       | Script endpoint from branding page  | `/_sec/verify?provider=adaptive` + script endpoint |
| Verify Endpoint   | `/_sec/cp_challenge/verify` (static) | Dynamic `verify_url` from challenge | `/_sec/cp_challenge/verify` (static)               |
| Success Indicator | `sec_cpt` contains `~3~`             | `sec_cpt` contains `~3~`            | `sec_cpt` contains `~3~`                           |
| Typical Flow      | Wait → POST PoW → Verify             | Fetch branding → Sensors → Verify   | Wait → POST PoW → Sensors → Verify                 |


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/akamai-web/handling-428-status-code-sec-cpt.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
