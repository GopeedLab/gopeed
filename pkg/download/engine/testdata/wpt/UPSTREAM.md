# Web Platform Tests provenance

The files under `fetch/` are copied from the official
[web-platform-tests/wpt](https://github.com/web-platform-tests/wpt) repository.

- Upstream commit: `ce5d9e7e28b27528213bceea40d9e78462487105`
- License: BSD-3-Clause; see `LICENSE.md`
- Canonical Fetch tests: `fetch/`

Gopeed executes applicable worker-style Fetch API tests at the extension
JavaScript boundary. The checked-in runner currently executes unchanged WPT
tests for Headers syntax and iteration, Request construction and body
consumption, ReadableStream bodies, keepalive, Request errors, Response
construction and body consumption, immutable network response headers, and
the Response static constructors.

Passing this subset is an extension-worker Fetch API compatibility claim, not
a claim that Gopeed is a browser. Browser-only suites that require a document,
CORS origin enforcement or preflight, CSP, Mixed Content, Service Workers,
browser HTTP cache, navigation, browser authentication UI, or WPT's
multi-origin server infrastructure are intentionally excluded. Gopeed also
keeps extension-specific behavior: `redirect: "manual"` exposes the HTTP
redirect response so download extensions can inspect `Location`, instead of
returning a browser `opaqueredirect` response.

Header permissions follow [Node/Undici's server-side Fetch API](https://github.com/nodejs/undici/blob/main/README.md#forbidden-and-safelisted-header-names). Gopeed does not
apply browser forbidden request/response header lists or no-CORS safelists.
Extensions may supply authentication, referrer, origin, host, proxy and fetch
metadata headers, and read `Set-Cookie` response headers. Network response
headers remain immutable; invalid header syntax still raises an error.
Repeated `Cookie` values are combined with `; `, as in Node/Undici.

The runner excludes the browser-only `headers-no-cors.any.js` and
`headers-forbidden-override.any.js` suites, the filtering assertions within
`request-headers.any.js`, and the forbidden-response-cookie assertion within
`header-setcookie.any.js`. Gopeed's `extension_headers_test.go` replaces those
restrictions with integration tests that check headers actually received by
an HTTP server through fetch and XMLHttpRequest, plus header mutations,
cloning and response cookies. The upstream test files remain unchanged.

Gopeed deliberately ignores transport controls that Node/Undici rejects:
`Expect`, `Keep-Alive`, `Transfer-Encoding`, `Upgrade`, and `Connection` values
other than `close` or `keep-alive`. `Content-Length` is generated from the
actual request body, ignoring a supplied length. Explicit `Host` and
`Sec-Fetch-Mode` values are preserved rather than replaced by defaults.
These transport behaviors are also covered by integration tests.
`credentials: "omit"` disables automatic CookieJar cookies but preserves
explicit headers; Gopeed retains its extension CookieJar support.

`wpt_harness.js` is a small Goja adapter maintained by Gopeed; it is not copied
from WPT. Test logic below `fetch/` is unchanged; the checked-in copy may only
normalize a final newline.
