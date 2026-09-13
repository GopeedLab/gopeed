# Extension browser host configuration

The downloader wraps each provider with the extension's stable identity before
constructing the runtime. `OpenOptions.ProfileID`, `DataPath`, and `ProxyURL` are
host-only fields. `parseOpenOptions` never reads them from extension JavaScript.
The extension API is unchanged.

Profiles are persistent and shared across invocations/pages of the same extension.
Different identities use different native stores. Go providers use a SHA-256-based
path under `<StorageDir>/webview`; Apple RPC hosts use a stable UUID data store;
Android uses a named WebView profile. Cookie operations select that same store,
including `clearCookies`. Closing an execution closes its pages, not its profile.
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
- In webview_go: `go run ./examples/profilecheck` (Linux: `xvfb-run -a ...`).
- In ui/flutter: `flutter run -d macos -t tool/webview_profile_check.dart`.

The native smoke tests cover routing and persistent profile isolation. The Flutter
smoke test also checks concurrent pages, HttpOnly cookies, LocalStorage, IndexedDB,
profile-scoped clearing, and reopening. The self-signed TLS fixture is trusted only
by that test's callback; production certificate validation is unchanged.
