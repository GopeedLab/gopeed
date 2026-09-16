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
