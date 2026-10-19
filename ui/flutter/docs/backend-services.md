# Backend services and protocol jobs

External metadata and protocol requests run in Go for Web, desktop FFI, and gomobile.
Native update packages are downloaded by the client to preserve byte progress. The
current release builds use `ui/flutter`; `ui/flutter_old` is an archived client
and is not part of the shipping workflow. The retired `/api/web/proxy` always
returns 404, including when static Web hosting is enabled.

## Protocol configuration and backend state

Clients use only `GET /api/v1/config` and `PUT /api/v1/config`. Protocol-specific
configuration GET/PATCH routes and capability RPCs have been removed. The common
`POST /api/v1/protocols/{protocol}/jobs/{name}/trigger` action remains available.

Protocols reuse `DownloaderStoreConfig.ProtocolConfig` and their config structs.
A manager can embed `fetcher.ConfigState[T]` to expose a typed, persisted `state`
section. This reserved section belongs to Go. Before decoding a config PUT,
Downloader replaces all supplied protocol state with the latest stored state
under the config lock. Omitted, null, forged, stale, or wrongly typed state is
ignored. Unregistered protocols, unknown config fields, and wrong writable-field
types are rejected. Optional protocol normalizers own business validation and
state cleanup (for example, pruning removed subscriptions).

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
`customTrackers`. Users can add, edit, and remove arbitrary HTTP(S) subscription
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
flags. The official URL is always included. `github.com`,
`raw.githubusercontent.com`, and `api.github.com` expand into eligible mirror
candidates; non-GitHub URLs use their original endpoints. jsDelivr only applies
to Raw files; GitHub API requests can use ghProxy mirrors.

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
  attempt has a fresh checkout and timeout. The canonical repository URL and
  subdirectory are preserved.

## Dedicated business APIs

- `POST /api/v1/app/updates/check`: `{currentVersion}`; preserves stable/preview
  version selection. The original GitHub Releases API and eligible ghProxy mirrors
  race as the first data provider. Only if all candidates fail does the independent
  gopeed.com provider run. Invalid JSON and non-release responses count as failures;
  a successful empty/no-update result does not trigger provider fallback.
- `POST /api/v1/app/updates/candidates`: `{version, asset}`; validates a fixed
  Gopeed release asset and returns `{urls: [...]}` containing official and eligible
  mirror URLs from the backend config. It accepts no arbitrary URL. Native clients
  probe candidates concurrently with bounded HEAD requests, prefer the fastest
  usable response, and download using Dio's real byte-progress callback. A failed
  probe remains a fallback because some servers do not support HEAD. Failed GETs
  and HTML/JSON landing pages fall back to the next candidate, removing partial
  files. Installation continues through the existing native updater/APK installer.
  Probe latency is a heuristic, not a measurement of sustained download speed.
- `GET /api/v1/store/extensions`: validated page/limit/sort/order/q parameters.
- `GET /api/v1/store/extensions/{id}`: optional version parameter.
- `POST /api/v1/store/extensions/{id}/install-report`: non-blocking from the UI.

All routes live in `pkg/api.Service`. Frontend operations use the shared
GopeedService capability descriptors and central codec registry. No frontend
GitHub URL rewriting, subscription fetching, or generic backend HTTP forwarding
remains in the active client. Package probing/downloading is a generic native
Dio helper in `core/network/candidate_download.dart` and uses backend-provided
URLs without reading mirror settings.
