# Getting started

This page covers various best practices. If you already have a lot of experience with Incapsula you can skip this and head directly to the [API Reference](/api-reference/incapsula.md).

## Which cookies does my site have?

### Utmvc

If your site loads in a script that looks like the following, then it has utmvc.

```
/_Incapsula_Resource?SWJIYLWA=...
```

More info here: [UTMVC](/incapsula/utmvc.md)

### Reese84

If your browser contains a cookie named "reese84" on the website or an `x-d-token`header, then the website uses reese84.

More info here: [Reese84](/incapsula/reese84.md)

### Complete Example

For a full working implementation with proper TLS client setup, header ordering, and cookie handling, see our examples repository:

{% embed url="<https://github.com/Hyper-Solutions/hypersolutions-examples>" %}


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/incapsula/getting-started.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
