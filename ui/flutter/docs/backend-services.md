# Backend services and protocol jobs

Protocol requests run in Go for Web, desktop FFI, and gomobile. Application
update checks and extension store requests run in the client. The Downloader
only supplies mirror candidate URLs for client requests; it does not fetch or
interpret application releases. Website requests go directly to gopeed.com.
Native update packages are downloaded by the client to preserve byte progress. The
current release builds use `ui/flutter`; `ui/flutter_old` is an archived client
and is not part of the shipping workflow. All unmatched `/api` and `/api/**`
requests use the common 404 handler before the web file handler, including the
removed `/api/web/proxy` endpoint. There are no proxy-specific compatibility routes; configured API authentication still applies.

## Protocol configuration and backend state

Clients use only `GET /api/v1/config` and `PUT /api/v1/config`. Protocol-specific
configuration GET/PATCH routes and capability RPCs have been removed. The common
`POST /api/v1/protocols/{protocol}/jobs/{name}/trigger` action remains available.

Protocols reuse `DownloaderStoreConfig.ProtocolConfig` and their config structs.
A manager can embed `fetcher.ConfigState[T]` to expose a typed, persisted `state`
section. This reserved section belongs to Go. Before decoding a config PUT,
Downloader replaces all supplied protocol state with the latest stored state
under the config lock. Omitted, null, forged, stale, or wrongly typed state is
ignored. Unknown config fields are ignored during both startup and config PUT,
so removed fields do not prevent upgrades from starting. Unregistered protocols
and wrong writable-field types are rejected. Optional protocol normalizers own
business validation and state cleanup (for example, pruning removed subscriptions).

`fetcher.ReadConfig[C]` returns a detached config snapshot, including state.
`fetcher.UpdateState[C, S]` receives the latest config and mutable state within
one locked transaction. Only the state is committed; edits to the config
snapshot are discarded. The result is normalized, persisted, then published.
Callback, validation, and storage failures leave the current config unchanged.
Callbacks must not perform network I/O or re-enter configuration operations.
State commits do not wake cron loops as if the user changed settings.

```go
type trackerState struct {
    LastTrackerUpdateTime time.Time `json:"lastTrackerUpdateTime"`
}
type config struct {
    fetcher.ConfigState[trackerState]
    AutoUpdateTrackers bool `json:"autoUpdateTrackers"`
}

// Fetch outside this callback. It only checks current config and assigns state.
err := fetcher.UpdateState[config, trackerState](host,
    func(current *config, state *trackerState) error {
        state.LastTrackerUpdateTime = time.Now().UTC()
        return nil
    })
```

Settings saves are serialized and use a single full config PUT. Failed edits
remain retryable; pending saves keep their provider alive until persistence
finishes. Concurrent full PUTs retain last-writer-wins semantics for writable
settings while preserving the latest backend state.

## Protocol jobs

`CronProvider.CronJobs(ProtocolHost)` registers names unique within the manager's
`Name()`. Each job declares an interval, timeout, execution function, and receives
a manual/scheduled trigger source. There is no separate Enabled callback:
protocols decide whether scheduled work should run. Downloader owns persisted
execution times, so registration does not seed a second scheduling timestamp.

`POST /api/v1/protocols/{protocol}/jobs/{name}/trigger` synchronously waits for the
job. It accepts no arbitrary URL, script, or scheduling parameters. Concurrent
calls for the same job share the in-flight run. A manual request that joins a
skipped scheduled run executes again as manual work. Canceling one waiter does
not cancel shared work. Manual failures also respect the scheduled retry backoff. Downloader owns job contexts, timeout, panic recovery,
bounded retry backoff, persisted success/failure state, startup catch-up, and
shutdown cancellation/draining. Config changes wake jobs after unlocking.

## BT Tracker subscriptions

The BT config contains writable `trackerSubscribeUrls`, `autoUpdateTrackers`, and
`customTrackers`. Users can add and remove arbitrary HTTP(S) subscription
URLs. `state.trackerSources`, `state.trackerSubscriptionStatuses`, and
`state.lastTrackerUpdateTime` are backend-owned results. Effective Trackers are
computed in Go from custom Trackers and the cached sources; duplicate derived
lists are not stored. Existing BT settings remain in the full config API.

The `bt/update-trackers` job runs every 24 hours. An overdue startup runs once;
a recently successful job waits until due. Scheduled runs honor the automatic
update flag; manual runs are allowed while automatic updates are off. Fetching
uses bounded concurrency, response limits, public target validation, and Go's
GitHub mirror racer. Responses must contain valid Tracker URLs to win the race.

Each source persists its last fetch time, success flag, and deduplicated Tracker
count from that attempt (zero on failure), including when all sources fail.
Sources without a status have never been fetched. Removed subscriptions lose
both their status and cached URLs. The client reloads these results after either
a successful or failed manual run.

Successful sources replace their caches; failed sources keep theirs. A partial
refresh commits usable data but reports failure for retry. Only a fully
successful refresh advances `state.lastTrackerUpdateTime`. If subscriptions change
while fetching, the obsolete batch is discarded and rescheduled. Derived lists
use the latest custom Trackers. Removed sources are pruned; explicitly empty
subscriptions remain empty. Active public torrents receive additions; private
torrents and unresolved torrent metadata are excluded from this update pass.

This release uses the new configuration layout without migrating old settings.
BT settings live in protocol config and GitHub mirror settings live in the
top-level downloader `githubMirror`. Missing fields are initialized in Go;
explicit false/empty values in the current schema are preserved.

## GitHub mirrors

Top-level `githubMirror` retains `enabled`, mirror type/URL, built-in, and deleted
flags. The official URL is always included. `github.com` and
`raw.githubusercontent.com` expand into eligible mirror candidates; other URLs,
including `api.github.com`, use their original endpoints. jsDelivr only applies
to Raw files. Defaults include `fastly.jsdelivr.net/gh`, `fastgit.cc`,
`gh-proxy.com`, and `ghfast.top`; all are marked as built-in.

`internal/githubmirror` owns configuration types/defaults, URL candidate generation,
and the shared generic `Race[T]` and `Fallback[T]` executors. Callers supply a
snapshot of the mirror config, the original URL, a resource/repository kind,
per-attempt timeout, and an operation callback. `Options[T].Discard` releases
partial failures and race losers; ownership of the winner transfers to the
caller. Execution drains canceled attempts before returning. Callbacks must
honor cancellation and return completed results rather than live response
streams. HTTP metadata fetching and extension Git operations use these same
executors. Native application downloads use this module's candidate generator
and let the client probe and download locally. See the [package usage guide](../../../internal/githubmirror/README.md).

- HTTP resources race official and eligible mirror requests. The first complete,
  validated result wins and cancels losers. jsDelivr rewrites raw source URLs
  while preserving the branch/commit; ghProxy prefixes the original URL.
- Extension Git installation, update checks, and updates try the official clone
  first and then ghProxy mirrors in order. jsDelivr is not a Git remote. Every
  attempt has a fresh checkout. Git uses a 10-second connection/TLS handshake
  timeout and a 60-second network read idle timeout, with no fixed clone duration.
  Continuous reads reset the idle allowance; stalled sources fall back in order.
  The canonical repository URL and subdirectory are preserved.

## Mirror candidates and client updates

`POST /api/v1/mirrors/{provider}/candidates` accepts `{url}` and returns
`{urls: [...]}`. Unknown providers return 404. Downloader dispatches through its
provider table; each implementation validates its own resources and reads its
own configuration. Adding a provider does not change this route or the shared
capability RPC. Currently `github` is the only provider and uses the existing
Downloader `githubMirror` config.

The GitHub provider accepts only HTTPS URLs on `github.com` and
`raw.githubusercontent.com`, without userinfo, ports, or fragments. API URLs
(`api.github.com`) are not supported. Invalid input
returns 400; configuration read failures return 500. The official URL is first,
followed by eligible mirrors in configured order, without duplicates. Disabled
mirrors return only the official URL. This API never fetches, probes, or forwards
requests. It replaces the removed `/api/v1/app/updates/check` endpoint.

`MirrorClient(provider: ..., service: ...).getJson<T>(url, decode: ...)` is the
reusable client-side racer. It gets candidates for the selected provider through
`GopeedService`, then uses a separate Dio client with no Downloader authentication
headers. Each metadata request has a 30-second
budget. The first complete response accepted by the caller's decoder wins;
losing requests are canceled. HTML, invalid JSON, and non-release responses
cannot win an update check.

`AppUpdateClient` requests the fixed official GitHub Releases URL directly,
without calling Downloader or the mirror candidates API. Only if this request
or its response validation fails does it request
`https://gopeed.com/api/releases?per_page=10` directly. A valid empty list or
no-update result does not trigger provider fallback. The GitHub request has a
30-second total budget and is canceled before fallback on timeout. Stable
versions ignore previews; previews allow newer previews in the same version
series and newer stable releases. Drafts and invalid versions are ignored.
Version selection is client-side, including with a remote Downloader. Release
assets retain `digest: sha256:<64 hex characters>` from these trusted APIs in an
immutable map keyed by exact package filename. Mirrors never supply expected
hashes. Native metadata requests also disable HTTP redirects.

When the user starts an update, the client selects the package name from its own
channel and architecture, and asks the same mirror candidates API with
`provider=github` for the asset URL. Native clients probe candidates using bounded concurrent HEAD requests and
prefer the fastest usable response. Failed probes remain fallbacks because some
servers do not support HEAD. Failed GETs and HTML/JSON landing pages fall back
to the next candidate, removing partial files. Dio reports actual download byte
progress; installation continues through the native updater/APK installer.
The selected package must have a valid official SHA-256 before any package
request starts. After each complete download, `candidate_download.dart` streams
the local file through SHA-256 and compares it with that trusted digest. A hash
mismatch deletes the file and tries the next candidate; if all fail, installation
is blocked. Missing or malformed official digests also stop automatic updates.
Download response headers cannot replace the trusted expected hash. Installation
is reached only after this verification returns successfully. The UI displays a
localized verification error when integrity checks fail.

Unsupported/store channels retain their installer or release-page flow. Probe
latency is a heuristic, not a sustained download speed measurement.

The website Worker adds CORS to `/api/releases` for credential-free Web requests,
including error responses and OPTIONS preflight. Deploy that website change
before releasing the new client fallback.

## Direct extension store requests

`GopeedSiteApi` uses its own Dio client for these fixed website endpoints:

- `GET https://gopeed.com/api/extensions`: page/limit/sort/order/q, with
  `view=summary`. Successful pages retain the existing bounded two-minute cache
  and concurrent request coalescing; failures remain retryable.
- `GET https://gopeed.com/api/extensions/{id}`: encoded ID and optional version
  query parameter. Detail providers retain their existing retries and cache.
- `POST https://gopeed.com/api/extensions/install`: `{id}` as JSON. This
  best-effort report does not block installation.

These requests never carry a Downloader API token. There are no store routes in
Downloader or store capability RPCs. The website's Cloudflare Worker supplies
CORS for these public endpoints, including cached responses, errors, and JSON
POST preflight. Deploy the website CORS change before releasing the direct Web
client. Extension Git installation and updates still execute in Go.

Downloader routes live in `pkg/api.Service`. Their frontend operations use the
shared GopeedService capability descriptors and central codec registry. No frontend
GitHub URL rewriting, subscription fetching, or generic backend HTTP forwarding
remains in the active client. Package probing/downloading is a generic native
Dio helper in `core/network/candidate_download.dart` and uses backend-provided
URLs without reading mirror settings.
