# Supported User Agents

### Desktop User Agents

#### Windows Chrome (Recommended)

```
Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{REPLACETHIS}.0.0.0 Safari/537.36
```

Windows remains our recommended default for desktop configurations.

#### macOS Chrome

```
Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{REPLACETHIS}.0.0.0 Safari/537.36
```

macOS support is available for specific use cases where Windows user agents may face temporary restrictions.

### Mobile User Agents

The Kasada API also supports the following mobile app user agents:

```
FootLocker/X.X.X iOS/X.X
ChampsSports/X.X.X iOS/X.X
KidsFootLocker/X.X.X iOS/X.X
FLCA/CFNetwork/Darwin
FLEU/CFNetwork/Darwin
Whatnot vXX.XX.0 (XX), iOS X.X, iPhoneXX,X
SNKRS/X.X.X (prod; XXXXXXXXXX; iOS X.X; iPhoneX,X)
NikeApp/XX.XX.X (prod; XXXXXXXXXX; iOS X.X; iPhoneX,X)
Sephora X.X, iOS XX.X.X, iPhoneXX,X
```

**Note:** The app version (e.g., `X.X.X`) and iOS version (e.g., `iOS X,X`) will change over time. Ensure you are using current versions for the best results.

#### X-Kpsdk-Dv header value for mobile

This value is hardcoded and we can't return it to you for the mobile endpoints. You can use the following strings:

```
QkZWEmcDRUBEDloaAg8GABpSDxVEX1JfXBRZRQF5VRoJFFozUQtXBABQGwEPHA==
QkZWEmcDRUBEDloaAg8GABpRDxVEX1JfXBRZRQF5VRoJFFozUQtXBABRGwEPHA==
```


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/k4sada/supported-user-agents.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
