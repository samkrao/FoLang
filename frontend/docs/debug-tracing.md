# Debug tracing

Parser and lexer tracing is disabled by default. To enable it, create
`folang-debug.json` in the working directory from which `fo-frontend` is
started:

```json
{
  "debug": {
    "trace": {
      "lexer": true,
      "parser": true
    }
  }
}
```

The file is read once, at process startup. A missing file or omitted setting
means `false`. Invalid JSON and unknown property names cause startup to fail
with an error on stderr. Trace output is also written only to stderr, keeping
LSP JSON-RPC output on stdout intact.
