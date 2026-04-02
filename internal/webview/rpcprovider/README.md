# rpcprovider

`rpcprovider` is the RPC-backed implementation of Gopeed's `webview.Provider` interface.

It is intended for environments where WebView capability is owned by another process, most notably mobile hosts. This package does not implement a WebView itself. Instead, it translates the `pkg/download/engine/webview` page API into single-request `POST + JSON` calls sent to a local host-side RPC service.

## When to use it

Use `rpcprovider` for entrypoints such as `bind/mobile`, where the Go process cannot directly own a native WebView.

Do not use it for desktop or `cmd/web` local WebView integration. Those entrypoints should use `internal/webview/goprovider` instead.

## Configuration

The provider is configured through `pkg/download/engine/webview.RPCConfig`:

```go
type RPCConfig struct {
	Network string `json:"network"`
	Address string `json:"address"`
	Token   string `json:"token,omitempty"`
}
```

Notes:

- `Network` currently supports `tcp` and `unix`.
- `Address` examples:
  - `127.0.0.1:38765`
  - `/path/to/webview.sock`
- `Token` is optional. When set, the client sends `Authorization: Bearer <token>`.

Example:

```go
provider := rpcprovider.New(webview.RPCConfig{
	Network: "tcp",
	Address: "127.0.0.1:38765",
	Token:   "secret-token",
})
```

## Transport

- Endpoint: `POST /webview`
- Content type: `application/json`
- One request, one response
- No `id` field yet
- The envelope is intentionally close to JSON-RPC so it can evolve later without renaming the methods or reshaping the payloads

Request example:

```json
{
  "method": "page.execute",
  "params": {
    "pageId": "page-1",
    "expression": "document.title",
    "args": []
  }
}
```

Successful response:

```json
{
  "result": {
    "title": "Example"
  },
  "error": null
}
```

Failure response:

```json
{
  "result": null,
  "error": {
    "code": "PAGE_NOT_FOUND",
    "message": "page not found"
  }
}
```

## Supported methods

- `webview.isAvailable`
- `page.open`
- `page.addInitScript`
- `page.navigate`
- `page.execute`
- `page.getCookies`
- `page.setCookie`
- `page.deleteCookie`
- `page.clearCookies`
- `page.close`

The canonical wire models for these methods are defined in:

- [pkg/download/engine/webview/rpc.go](../../../pkg/download/engine/webview/rpc.go)
- [pkg/download/engine/webview/runtime.go](../../../pkg/download/engine/webview/runtime.go)

The README is only a guide. The Go source above is the authoritative protocol reference.

## Method reference

Each method below uses the common envelope:

```json
{
  "method": "<method-name>",
  "params": { ... }
}
```

and returns:

```json
{
  "result": ...,
  "error": null
}
```

or:

```json
{
  "result": null,
  "error": {
    "code": "<ERROR_CODE>",
    "message": "<message>"
  }
}
```

### `webview.isAvailable`

Request:

```json
{
  "method": "webview.isAvailable",
  "params": {}
}
```

Success:

```json
{
  "result": {
    "available": true
  },
  "error": null
}
```

### `page.open`

Request:

```json
{
  "method": "page.open",
  "params": {
    "headless": true,
    "debug": false,
    "title": "Gopeed WebView",
    "width": 1280,
    "height": 720,
    "userAgent": "Mozilla/5.0 ..."
  }
}
```

Success:

```json
{
  "result": {
    "pageId": "page-1"
  },
  "error": null
}
```

### `page.addInitScript`

Request:

```json
{
  "method": "page.addInitScript",
  "params": {
    "pageId": "page-1",
    "script": "window.__READY__ = true;"
  }
}
```

Success:

```json
{
  "result": {},
  "error": null
}
```

### `page.navigate`

Request:

```json
{
  "method": "page.navigate",
  "params": {
    "pageId": "page-1",
    "url": "https://example.com",
    "timeoutMs": 15000
  }
}
```

Success:

```json
{
  "result": {},
  "error": null
}
```

### `page.execute`

Request:

```json
{
  "method": "page.execute",
  "params": {
    "pageId": "page-1",
    "expression": "document.title",
    "args": []
  }
}
```

Success:

```json
{
  "result": "Example Domain",
  "error": null
}
```

### `page.getCookies`

Request:

```json
{
  "method": "page.getCookies",
  "params": {
    "pageId": "page-1"
  }
}
```

Success:

```json
{
  "result": [
    {
      "name": "session",
      "value": "abc",
      "domain": ".example.com",
      "path": "/",
      "expires": "2026-03-26T12:00:00Z",
      "secure": true,
      "httpOnly": true
    }
  ],
  "error": null
}
```

### `page.setCookie`

Request:

```json
{
  "method": "page.setCookie",
  "params": {
    "pageId": "page-1",
    "cookie": {
      "name": "session",
      "value": "abc",
      "domain": ".example.com",
      "path": "/",
      "expires": "2026-03-26T12:00:00Z",
      "secure": true,
      "httpOnly": true
    }
  }
}
```

Success:

```json
{
  "result": {},
  "error": null
}
```

### `page.deleteCookie`

Request:

```json
{
  "method": "page.deleteCookie",
  "params": {
    "pageId": "page-1",
    "cookie": {
      "name": "session",
      "domain": ".example.com",
      "path": "/"
    }
  }
}
```

Success:

```json
{
  "result": {},
  "error": null
}
```

### `page.clearCookies`

Request:

```json
{
  "method": "page.clearCookies",
  "params": {
    "pageId": "page-1"
  }
}
```

Success:

```json
{
  "result": {},
  "error": null
}
```

### `page.close`

Request:

```json
{
  "method": "page.close",
  "params": {
    "pageId": "page-1"
  }
}
```

Success:

```json
{
  "result": {},
  "error": null
}
```

## Method semantics

- `page.open`
  - Creates a page session
  - Returns `pageId`
- `page.addInitScript`
  - Registers a page initialization script
- `page.navigate`
  - Navigates to the target URL
  - The server should return only after navigation completes or fails
- `page.execute`
  - Executes an expression or serialized function body
  - The returned value must be JSON-serializable
- `page.getCookies`
  - Returns cookies from the underlying native WebView cookie store
- `page.setCookie`
  - Inserts or updates a cookie
- `page.deleteCookie`
  - Deletes a cookie using `name/domain/path`
- `page.clearCookies`
  - Clears the current cookie store
- `page.close`
  - Closes the page and releases related resources

## Cookie model

Cookies use `runtime/webview.Cookie`.

Example payload:

```json
{
  "name": "session",
  "value": "abc",
  "domain": ".example.com",
  "path": "/",
  "expires": "2026-03-26T12:00:00Z",
  "secure": true,
  "httpOnly": true
}
```

Notes:

- `expires` uses RFC3339 / RFC3339Nano string form.
- If the host platform supports a full native cookie store, it should return fields such as `httpOnly` and `secure`.
- `setCookie -> navigate` is a valid pattern for preloading authenticated session state.

## Error codes

The current error code set is defined in `pkg/download/engine/webview/rpc.go`:

- `INVALID_REQUEST`
- `UNKNOWN_METHOD`
- `UNAVAILABLE`
- `BROWSER_NOT_FOUND`
- `PAGE_NOT_FOUND`
- `NAVIGATION_FAILED`
- `EVALUATION_FAILED`
- `TIMEOUT`
- `INTERNAL_ERROR`

`BROWSER_NOT_FOUND` is a historical leftover from an earlier shape of the protocol. The current protocol no longer has a separate browser model, so this can be cleaned up later if the protocol is tightened further.

## Current protocol boundary

This protocol is intentionally limited:

- Single-direction request/response only
- Not REST-style resource routing
- No bidirectional events
- No streaming event subscriptions
- No host-initiated push channel

If the protocol needs to evolve toward full JSON-RPC later, the intended migration path is to keep the existing `method/params/result/error` structure and add:

- `id`
- notifications
- a bidirectional transport model

## Host implementation guidance

A host implementation should usually:

- Listen on a local `tcp` address or `unix socket`
- Expose `POST /webview`
- Validate the bearer token if configured
- Maintain `pageId -> native webview/session`
- Own page lifecycle, cleanup, and timeout handling

With that split, Gopeed stays a pure RPC client and does not need to know how the host's native WebView is implemented.
