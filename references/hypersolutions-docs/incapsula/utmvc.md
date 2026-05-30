# UTMVC

The utmvc script path needs to be parsed from the HTML of the page. You can do this by using the regular expression:

```regex
src="(/_Incapsula_Resource\?[^"]*)"
```

You then need to make a request to the script to obtain the JavaScript code, and send the JavaScript to the API. You can then set a cookie named "\_\_\_utmvc" (note: three underscores) with the response from the API.

\
Then, you need to submit a GET request to /\_Incapsula\_Resource?SWKMTFSR=1\&e=. The value of e should be a random 64-bit floating point. For example, the full path will be /\_Incapsula\_Resource?SWKMTFSR=1\&e=0.14896897949050825. Once you have done this, assuming the utmvc cookie is valid, the server will set the utmvc cookie to the value of "a" with a max age of zero.\
After this, you can make requests to the site.


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://docs.hypersolutions.co/incapsula/utmvc.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.
