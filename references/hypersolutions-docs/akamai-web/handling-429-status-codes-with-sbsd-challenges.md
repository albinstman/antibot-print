# Handling 429 Status Codes with SBSD Challenges

### Understanding 429 SBSD Blocks

When interacting with APIs protected by SBSD, you may occasionally encounter a `429 Too Many Requests` status code. Instead of the expected API response, you'll receive a JSON response containing a challenge token:

```json
{
  "t": "183446612"
}
```

This indicates that the SBSD protection system has triggered a challenge that must be solved before you can continue making requests to the API.

### Solution Process

Solving a 429 SBSD block follows a similar process to the standard SBSD challenge flow explained in our [SBSD Challenge Flow Documentation](/akamai-web/sbsd-challenge-flow.md), but with a simplified approach:

#### Step 1: Extract the Challenge Token

From the 429 response, extract the `t` value (challenge token):

```javascript
const response = await fetch("https://example.com/api/resource");
if (response.status === 429) {
  const data = await response.json();
  const challengeToken = data.t;  // In our example: "183446612"
  
  // Proceed to solve the challenge
}
```

#### Step 2: Construct the Challenge URL Path

For a 429 response, you'll need to use the same script path and `v`parameter as in previous successful SBSD solves. You must store this information before making the API request. You can also reuse the script content you have requested previously.

#### Step 3: Generate a New Payload

Use our API to generate a fresh SBSD payload:

```javascript
const payload = await generatePayload({
  userAgent: "YOUR_USER_AGENT",
  uuid: "YOUR_STORED_UUID",  // Use the UUID from a previous challenge
  pageUrl: "https://example.com/",
  o: getCookie("sbsd_o") || getCookie("bm_so"),
  script: scriptContent,
  ip: yourIp,
  acceptLanguage: yourAcceptLanguage
});
```

#### Step 4: Submit the Solution

POST the generated payload to the challenge endpoint:

```javascript
const submitUrl = `https://example.com${scriptPath}?t=${challengeToken}`;
await fetch(submitUrl, {
  method: "POST",
  headers: {
    "Content-Type": "application/json"
  },
  body: JSON.stringify({ body: payload })
});
```

#### Step 5: Retry Your Original API Request

Once the challenge is solved, retry your original API request. It should now proceed normally:

```javascript
const retryResponse = await fetch("https://example.com/api/resource");
// Process the successful response
```

### Complete Example

Here's a pseudocode example showing how to handle a 429 SBSD challenge in your API requests:

```javascript
async function fetchWithSbsdHandling(url, options = {}) {
  let response = await fetch(url, options);
  
  // Check if we received a 429 with a challenge token
  if (response.status === 429) {
    try {
      const data = await response.json();
      
      if (data.t) {
        // Extract the challenge token
        const challengeToken = data.t;

        // Generate the payload using our API
        const payload = await generatePayload({
          userAgent: options.headers['User-Agent'],
          uuid: getStoredUuid(),
          pageUrl: new URL(url).origin,
          oCookie: getCookie("sbsd_o") || getCookie("bm_so"),
          script: scriptContent,
          ip: yourIp,
          acceptLanguage: yourAcceptLanguage
        });
        
        // Submit the solution
        const submitUrl = `${new URL(url).origin}${scriptPath}?t=${challengeToken}`;
        await fetch(submitUrl, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "User-Agent": options.headers['User-Agent'],
            ... more headers
          },
          body: JSON.stringify({ body: payload })
        });
        
        // Retry the original request
        return fetch(url, options);
      }
    } catch (error) {
      console.error("Failed to solve SBSD challenge:", error);
    }
  }
  
  return response;
}
```

### Implementation Best Practices

1. **Store Challenge Information**: Save script paths and UUIDs from previous challenges.
2. **Automatic Retries**: Implement automatic SBSD challenge solving and request retries in your API client.
3. **Consistent Headers**: Maintain the same headers and match header order throughout the entire challenge solving process.
4. **Session Management**: Properly store and forward cookies between requests to maintain session state.

### Integrating with Our API

Our API service simplifies the payload generation process.&#x20;

For detailed API integration instructions and complete documentation on our payload generation service, refer to our [API Reference Documentation](/api-reference/akamai.md).


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/akamai-web/handling-429-status-codes-with-sbsd-challenges.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
