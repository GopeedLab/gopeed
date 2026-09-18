# Extension browser host configuration

The downloader wraps each provider with the extension's stable identity before
constructing the runtime. `OpenOptions.ProfileID`, `DataPath`, and `ProxyURL` are
host-only fields. `parseOpenOptions` never reads them from extension JavaScript.
The extension API is unchanged. The Flutter host uses the fork's public
`WebViewProfile`, `InAppWebViewSettings.profileId`, and profile-scoped
`CookieManager` APIs; the library has no Gopeed-specific method-channel bridge.
Only the Gopeed extension JavaScript boundary keeps these options host-only.

Profiles are persistent and shared across invocations/pages of the same extension.
Different identities use different native stores. Go providers use a SHA-256-based
path under `<StorageDir>/webview`; Apple RPC hosts use a stable UUID data store;
Android uses a named WebView profile. Cookie operations select that same store,
including `clearCookies`. Closing an execution closes its pages, not its profile.
Uninstall closes the extension's pages, prevents its old runtimes from opening
new pages, and removes its native profile before deleting the extension record.
Cleanup failures abort uninstall and can be retried. Reinstallation starts with
empty browser data; an ordinary extension update keeps the existing profile.
Windows/Linux delete the profile directory; Apple removes the named WebKit store.
Android cannot delete a profile already loaded by the current process: when
DELETE_BROWSING_DATA is supported it clears all site data immediately, and records
an empty-profile deletion in SharedPreferences for the next process start. A
reinstallation may reuse that cleared profile and cancels the pending deletion.
If the full-data clearing API is unavailable, uninstall fails with a restart
instruction; the pending profile is deleted at startup before it can be loaded.
No removal operation is exposed through extension JavaScript.

Existing shared browser data is not copied into every extension's new profile.

The downloader lazily owns one loopback forward proxy with HTTP and SOCKS5
listeners. Windows/Linux/Android use HTTP; Apple uses SOCKS5 because its HTTP
CONNECT configuration may bypass plain HTTP. The forwarder reads the current
Gopeed proxy config for each new request/tunnel and supports direct, system,
HTTP(S) upstream proxies with Basic authentication, and authenticated SOCKS5.
Credentials remain in Go and are never sent over the WebView RPC.

Changes affect new requests/connections. Existing CONNECT/SOCKS tunnels keep their
route until closed; opening another page in the same profile does not reset it.
SOCKS carries host/port rather than a full URL: system proxy/PAC selection for
Apple is host-based, with port 80 treated as HTTP and other ports as HTTPS.
Path-dependent PAC rules cannot be reproduced from a SOCKS connection.

Apple requires macOS 14 / iOS 17. Android checks MULTI_PROFILE and PROXY_OVERRIDE.
Unsupported hosts fail explicitly instead of silently sharing browser storage or
ignoring the proxy.

## Validation

- `go test -race ./internal/webview/proxy ./pkg/download/engine/webview`
- `go test ./pkg/download -run 'TestWebViewProfileBound|TestDownloader_ExtensionRuntimeWebView'`
- Windows/Linux: `go test -tags webview -v ./internal/webview/goprovider` (Linux: run under `xvfb-run -a`).
- macOS: start the normal Flutter app, then run `go test -tags webview -v ./internal/webview/rpcprovider` with its socket address. See `rpcprovider/README.md`.
- In webview_go: `go test -tags webview_integration -v -count=1 .` (Linux: run under `xvfb-run -a`).

The shared provider contract covers HTTP/HTTPS proxy routing, concurrent pages,
HttpOnly cookies, LocalStorage, IndexedDB, profile-scoped clearing, reopening,
and removal without affecting another profile. HTTPS routing is checked with a
rejecting upstream tunnel; no test-only certificate bypass is added to the app.
Go proxy unit tests separately cover successful TLS forwarding and authentication.
These cases run in the existing `test.yml` WebView jobs; existing mobile build jobs
compile the Android/iOS bridge. No separate Flutter test application is required.

## Extension events and host version

`gopeed.info.hostVersion` is the host's build version (`dev` for development
builds). `gopeed.info.version` continues to identify the extension's own version.
`gopeed.info.os` and `gopeed.info.arch` match the REST info endpoint's `os` and
`arch`: the host's Go `runtime.GOOS` and `runtime.GOARCH` values (for example,
`windows` / `amd64` or `darwin` / `arm64`).
See [extension-api.d.ts](extension-api.d.ts) for the event payloads.

```js
const page = await gopeed.runtime.webview.open();
const unsubscribe = await page.on("url-changed", ({ url, sameDocument }) => {
  gopeed.logger.info(url, sameDocument);
});
await page.on("closed", ({ reason }) => gopeed.logger.info(reason));
await page.addInitScript("globalThis.extensionReady = true");
await page.goto("https://example.com");
unsubscribe();
await page.close();
```

Registration resolves only after the listener is installed; it does not replay
current state. `url-changed` reports actual address changes, including hash,
pushState, replaceState and history navigation. `load` follows the main document's
`window.load`, including reloads, and does not wait for business-level async work.
Iframe loads and subresource failures do not become main-page notifications.
`closed` distinguishes the user's close action from API closure and is terminal.
Unsubscribe is idempotent. Notifications cannot block navigation; the engine does
not await promises returned by handlers. Catch async errors inside the handler.

`addInitScript` applies to future document contexts only. Use `execute` to run
code on the current page. Executable functions are serialized and cannot capture
extension-side lexical variables; pass values explicitly as arguments.

The RPC host uses an NDJSON `page.events` response whose first record is
`{"ready":true}`. Subsequent records are `{event,data}`. Closing a page sends the
terminal record before ending the stream. The native backend uses document-start
notifications for main-document URL/load events and native callbacks for
navigation failures and user closure.

Additional validation:

- `go test -race ./pkg/download/engine/webview ./internal/webview/rpcprovider`
- `go test ./pkg/download -run 'TestExtension(WebViewEvents|InfoHostVersion)'`
- macOS native: `go test -tags webview_native ./internal/webview/goprovider -run TestProviderEvents`
- Running Flutter host: `go test -tags webview ./internal/webview/rpcprovider -run TestProviderEvents`
