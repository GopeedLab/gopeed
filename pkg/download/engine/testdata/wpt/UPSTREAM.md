# Web Platform Tests provenance

The files under `fetch/` are copied from the official
[web-platform-tests/wpt](https://github.com/web-platform-tests/wpt) repository.

- Upstream commit: `ce5d9e7e28b27528213bceea40d9e78462487105`
- License: BSD-3-Clause; see `LICENSE.md`
- Canonical Fetch tests: `fetch/`

Gopeed executes applicable worker-style Fetch API tests at the extension
JavaScript boundary. The checked-in runner currently executes unchanged WPT
tests for Headers, no-CORS header guards, Request construction and body
consumption, ReadableStream bodies, keepalive, Request errors, Response
construction and body consumption, immutable network response headers, and
the Response static constructors.

Passing this subset is an extension-worker Fetch API compatibility claim, not
a claim that Gopeed is a browser. Browser-only suites that require a document,
CORS origin enforcement or preflight, CSP, Mixed Content, Service Workers,
browser HTTP cache, navigation, browser authentication UI, or WPT's
multi-origin server infrastructure are intentionally excluded. Gopeed also
keeps one extension-specific behavior: `redirect: "manual"` exposes the HTTP
redirect response so download extensions can inspect `Location`, instead of
returning a browser `opaqueredirect` response.

`wpt_harness.js` is a small Goja adapter maintained by Gopeed; it is not copied
from WPT. Test logic below `fetch/` is unchanged; the checked-in copy may only
normalize a final newline.
