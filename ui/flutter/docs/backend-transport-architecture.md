# Gopeed Go–Flutter In-Process API Architecture

## 1. Background

The Flutter native client currently starts the Go backend together with a REST server, then calls that REST API over TCP or a Unix socket. REST remains appropriate for the web client, but using it inside a native process introduces several problems:

- Flutter and Go run in the same process but still need to manage a listening port, Unix socket, API token, and connection fallback.
- The REST server lifecycle is coupled to the Downloader lifecycle, so the API server cannot be stopped without stopping the download core.
- Go already emits task events, but Flutter can only poll the task list instead of receiving in-process notifications.
- Flutter stores connection settings and preferences in Hive while Go uses Bolt Storage, creating two persistence systems.
- Adding an API can require duplicate work in the Go REST layer, native bridge, and Flutter client.

The goal is to let native clients call Go directly, retain REST for the web, and provide one consistent API experience to Flutter components.

## 2. Decision

Use the following platform strategy:

| Platform | Flutter-to-Go transport | Local REST server |
| --- | --- | --- |
| Windows / macOS / Linux | Dart FFI | Disabled by default; may be enabled separately |
| Android / iOS | MethodChannel + gomobile | Disabled by default; may be enabled separately |
| Web | REST | Required |

The objective is to remove socket transport from native business calls, not to force every native platform to use `dart:ffi`. Mobile platforms retain the existing gomobile AAR/XCFramework build pipeline because it has lower implementation and release risk.

Overall architecture:

```text
Flutter Widget / Riverpod Controller
                  |
             GopeedClient
                  |
          GopeedTransport
       +----------+-----------+
       |          |           |
 Desktop FFI   Mobile      Web REST
 Transport     gomobile    Transport
       |          |           |
       +----------+-----------+
                  |
            API Dispatcher
                  |
        Typed Application Service
                  |
       Downloader / Storage / Events
                  |
        Optional REST Server Adapter
```

## 3. Design Goals

### 3.1 Required outcomes

- Native business calls do not use TCP or Unix sockets.
- Web continues to use REST. All published core REST paths, HTTP methods, parameters, and response contracts remain fully compatible.
- The REST server can be started and stopped independently without closing the Downloader.
- Flutter components depend on one `GopeedClient` and do not branch on FFI, gomobile, or REST.
- Go can push task events to Flutter.
- Native clients use events for terminal task notifications; web may continue polling.
- Go Bolt Storage is the primary persistence layer for business configuration and application preferences.
- Flutter's Hive dependency is removed, with no migration of existing Hive data.
- Adding a normal API does not require separate changes to Go REST, Desktop FFI, and the mobile bridge.

### 3.2 Capabilities excluded from the unified business API

The following endpoints are web HTTP infrastructure and are not part of the FFI/gomobile API:

- `/fs/tasks/*`
- `/fs/extensions/*`
- `/api/web/proxy`
- Web static file hosting at `/`

`/fs/tasks` returns files from the Go host machine to a browser over HTTP. Native clients can access local file paths directly and do not need an equivalent API.

`/api/web/proxy` is infrastructure used by the web transport to work around browser CORS restrictions. It is not part of the published core `/api/v1/**` API. Native clients can request the target URL directly with Dio and do not need the Go proxy.

## 4. Backend Maintenance Model

### 4.1 Layers

The Go backend is divided into the following responsibilities:

```text
Runtime
├── ApplicationService
├── Dispatcher
├── TaskEventCallback
├── Storage
└── RestServer (optional)
```

#### Runtime

The Runtime owns the in-process singleton lifecycle:

- Initialize and close the Downloader.
- Initialize Bolt Storage.
- Create the ApplicationService, Dispatcher, and task-event callback adapter.
- Start, stop, or restart the REST server independently.
- Prevent hot reload or repeated initialization from creating multiple Downloader instances.

`Runtime.StopRestServer()` closes only the HTTP listener. `Runtime.Close()` closes the Downloader and Storage.

#### ApplicationService

This is the single business implementation layer. It exposes typed Go methods such as:

```text
Resolve
CreateTask
CreateTaskBatch
PatchTask
GetTasks
GetTaskStatus
PauseTasks
ContinueTasks
DeleteTasks
GetConfig
PutConfig
GetExtensions
...
```

Service methods must not depend on HTTP-specific concepts:

- `http.Request`
- `http.ResponseWriter`
- mux path variables
- HTTP headers

They accept business DTOs and return a consistent business result.

#### Dispatcher

The Dispatcher is the shared entry point for REST, FFI, and gomobile. Routes are registered explicitly instead of using Go reflection:

```text
RouteSpec
├── Method: GET
├── Path: /api/v1/tasks
└── Handler: service.GetTasks
```

The Dispatcher accepts a normalized request:

```json
{
  "method": "GET",
  "path": "/api/v1/tasks",
  "query": "status=running&status=done",
  "body": ""
}
```

It returns a normalized response:

```json
{
  "statusCode": 200,
  "body": {
    "code": 0,
    "data": []
  }
}
```

This format does not start HTTP inside a native client and does not pass through `net/http`. The `method/path/query/body` fields reuse the existing REST contract as an in-process request format.

This allows the same typed Flutter client method to work with REST, FFI, and gomobile without adding a separate native mapping for every API.

`reflect.MethodByName` is not allowed. Every route must be registered in an explicit registry that validates the following at startup:

- No duplicate method-and-path pairs.
- Valid path parameters.
- Non-null handlers.

#### REST adapter

The REST adapter is limited to transport responsibilities:

- Convert `http.Request` into a Dispatcher request.
- Write a Dispatcher response back to HTTP.
- Handle API tokens, web login, CORS, compression, and panic recovery.
- Register web-only file, proxy, and static resource routes.

The REST adapter no longer calls the global `rest.Downloader` directly.

### 4.2 Route groups

```text
Core API routes
├── /api/v1/info
├── /api/v1/tasks/*
├── /api/v1/config
├── /api/v1/extensions/*
└── /api/v1/runtime/*

Web-only routes
├── /api/web/login
├── /api/web/proxy
├── /fs/tasks/*
├── /fs/extensions/*
└── /* static Web UI
```

Core routes enter the Dispatcher and can be called through every transport.

Web-only routes are registered only by the REST server. They do not enter the FFI/gomobile ABI and do not require empty native implementations.

### 4.3 REST server configuration and runtime state

REST configuration is persisted in the `api` field of `DownloaderStoreConfig`:

```json
{
  "enable": false,
  "network": "tcp",
  "address": "127.0.0.1:9999",
  "token": ""
}
```

Rules:

- `DownloaderStoreConfig.api` represents desired configuration only. It does not prove that a listener is running.
- A fresh native installation defaults to `enable=false`. On startup, Go reads the persisted configuration and attempts to start the optional REST server when `enable=true`.
- A web server always enables REST; Flutter Web is only a REST client.
- A REST server enabled separately by a native client exposes the core API by default.
- The Flutter settings page always shows protocol, listen address, and token fields so they can be configured before startup.
- Saving configuration and starting, stopping, or restarting the service are separate backend operations. Flutter does not pass listener configuration directly to the listener manager and does not own persistence behavior.
- Save operations write through the config API. Lifecycle operations read the persisted desired configuration from Go Storage.
- Go keeps actual runtime state in memory: `running`, the active network/address/port, `pendingApply`, and `lastError`. These fields are not persisted.
- `pendingApply=true` means desired configuration and actual runtime state differ. This can happen after configuration is saved without a restart or after startup fails.
- Listener startup failure updates runtime state and error information without closing the Downloader, Storage, or in-process API service. If persisted `enable=true`, the next application startup attempts the listener again.
- Reconfiguration uses fast-fail semantics: stop the old listener, then start the new one. If startup fails, the listener remains stopped and the old listener is not restored. The new desired configuration remains persisted, while `pendingApply` and `lastError` report that it is not active.
- The web UI cannot stop the REST server that serves it.

Flutter native settings behavior:

- Show actual service status, the active listen address when running, `lastError` on failure, and a pending-apply message when configuration is not active.
- Keep configuration fields visible at all times. Do not use one switch to represent both persisted intent and runtime state.
- Provide separate Start/Stop and Save buttons. Allow buttons to wrap on narrow layouts.
- Saving while stopped only persists configuration.
- Saving while running validates the draft and asks whether to save and restart immediately. On confirmation, save first and then ask Go to restart.
- Stopping requires confirmation. On confirmation, persist `enable=false` and stop the listener.
- Starting persists `enable=true`, then asks Go to start the listener from persisted configuration.

## 5. Native ABI and Mobile Bridge

### 5.1 Desktop FFI

Keep the Desktop C ABI small and stable:

```text
Start
Stop
GetAPIServerState
StartAPIServer
StopAPIServer
RestartAPIServer
Invoke
FreeCString
SubscribeTaskEvents
```

Requirements:

- Use the cgo C-string ABI in the first phase. Reconsider pointer-and-length parameters only during a future ABI-breaking revision.
- Memory allocated by Go must be released with `FreeCString`.
- Dart-owned input memory must be released after the synchronous native call returns.
- `Invoke` returns the same Result JSON as REST; business errors do not use a separate ABI error model.
- Run blocking Desktop FFI work outside the Flutter UI isolate. Mobile MethodChannel handlers likewise run blocking Go work on a background task queue.
- Dart bindings are generated from `include/libgopeed.h` with `ffigen`. Every exported Desktop symbol must be added to the `ffigen.functions.include` allowlist.
- Never hand-edit `lib/core/ffi/libgopeed_bind.dart`; regenerate it with `flutter pub run ffigen` after changing the C header.

### 5.2 Android and iOS gomobile

Mobile continues to export:

```text
Start(configJson)
Stop()
GetAPIServerState()
StartAPIServer()
StopAPIServer()
RestartAPIServer()
Invoke(method, path, query, body)
SubscribeTaskEvents(mask, listener)
```

Flutter bridge rules:

- Normal calls use MethodChannel.
- Android performs normal calls on a background task queue.
- Events reuse the same MethodChannel through the reverse `taskEvent` message.
- Kotlin and Swift contain no business logic. They only forward arguments, dispatch callbacks to the UI thread, and translate errors.

## 6. Event Notification Design

### 6.1 Minimal task callback

The current requirement is limited to notifying native clients when a task completes or fails. Do not introduce an EventHub, durable queue, drain protocol, sequence numbers, or progress coalescing for these two low-frequency events.

The Downloader retains its existing single-listener mechanism. The shared API Service registers one adapter, and Flutter passes a bit mask describing the terminal events it wants Go to forward:

```text
TaskEventDone  = 1
TaskEventError = 2
```

Event payload:

```json
{
  "type": "task.done",
  "taskId": "task-id",
  "name": "example.zip",
  "error": null
}
```

The callback runs after the Downloader releases task locks. Desktop's `NativeCallable.listener` and the mobile UI-thread forwarding step return quickly and perform no UI or expensive business work inside the Go callback. Platform callback panics are isolated from download workers.

If high-frequency progress events or reliable multi-consumer delivery become necessary, introduce a queued EventHub as a separate design change instead of paying that complexity now.

### 6.2 Flutter event API

Flutter exposes the following to application components:

```text
Stream<TaskEvent> taskEvents
subscribeTaskEvents(Set<TaskEventType> events)
```

Implementation:

- Desktop: `NativeCallable.listener` receives JSON and calls `FreeCString` for the Go-owned payload.
- Mobile: a gomobile listener receives JSON and forwards it through MethodChannel.
- Web: the event stream is empty and task pages continue using their existing refresh strategy.

Components do not distinguish FFI from gomobile. The current desktop notification controller consumes `taskEvents` directly and replaces the previous two-second notification polling loop.

## 7. Flutter API Maintenance Model

### 7.1 Directory responsibilities

Use the following organization within the existing project structure:

```text
lib/core/network/gopeed/
├── gopeed_client.dart
├── gopeed_transport.dart
├── api_request.dart
├── api_response.dart
├── api_exception.dart
├── transports/
│   ├── rest_transport.dart
│   ├── desktop_ffi_transport.dart
│   ├── mobile_gomobile_transport.dart
│   └── transport_factory.dart
└── events/
    ├── gopeed_event.dart
    ├── event_source.dart
    └── task_event_reducer.dart
```

The existing `lib/api/model/` directory may remain initially to avoid a large model migration during the transport refactor.

### 7.2 Transports only transport data

Unified interface:

```dart
abstract interface class GopeedTransport {
  Future<ApiResponse> request(ApiRequest request);
  Stream<GopeedEvent> get events;
  Future<void> close();
}
```

A transport must not expose business methods such as `createTask()` or `getTasks()`. Otherwise, every new API would still require changes to all three transports.

### 7.3 Maintain one typed API in GopeedClient

All business methods are implemented once in `GopeedClient`:

```dart
Future<List<Task>> getTasks(List<Status> statuses) {
  return _request(
    method: 'GET',
    path: '/api/v1/tasks',
    query: {'status': statuses.map((e) => e.name).toList()},
    decode: decodeTasks,
  );
}
```

The same method is dispatched differently by platform:

- Web: RestTransport converts it into a Dio request.
- Desktop: FFI Transport converts it into `GopeedInvoke`.
- Mobile: gomobile Transport converts it into a MethodChannel invocation.

Components and Riverpod controllers inject only `GopeedClient` or a higher-level `GopeedService`.

The following patterns are prohibited:

- Feature widgets importing `api/api.dart` directly.
- Feature widgets accessing Dio, FFI, or MethodChannel directly.
- Components using `kIsWeb` to select a business API.
- Child windows initializing the Go Runtime.

### 7.4 Non-business HTTP capabilities

`proxyRequest` is not a Gopeed Core API and belongs in a separate `ExternalHttpClient`:

```text
Native → request the target directly with Dio
Web    → /api/web/proxy
```

Task file access does not belong in `GopeedClient`:

```text
Native → task.storagePath + relativePath
Web    → /fs/tasks/{taskId}/{relativePath}
```

A web resource helper generates resource URLs used by the web page.

### 7.5 Multiple windows

The main window is the sole owner of the Runtime, GopeedClient, and EventSource.

Child windows continue to use the existing AppCapabilities RPC:

- The main window forwards child requests to GopeedClient.
- The main window pushes complete state snapshots to child windows.
- Child windows do not load the dynamic library, call gomobile, or open Storage.

## 8. Cost of Adding an API

### 8.1 Normal query or command

For example, to add `GET /api/v1/tasks/{id}/peers`:

Backend work:

1. Add a typed method to ApplicationService.
2. Register its method, path, and handler in the Core Route Registry.
3. Add Service/Dispatcher contract tests.

Flutter work:

4. Add a Dart model and JSON codec if the API introduces a new data structure.
5. Add one typed method to `GopeedClient`.
6. Call that method from the feature Provider or Controller.

No changes are required in:

- REST Transport.
- Desktop FFI ABI.
- Android MainActivity.
- iOS AppDelegate.
- gomobile exports.
- Task-event bridging.

Typical effort:

| Change type | Backend effort | Flutter client effort | Transport effort |
| --- | --- | --- | --- |
| Simple query/command using existing DTOs | 15–40 lines + tests | 5–15 lines + tests | 0 |
| New request or response DTO | 30–80 lines + tests | Model/codec + 5–15 client lines | 0 |
| New batch operation | 30–100 lines + tests | 10–30 lines | 0 |
| New task-event type | Mask, filtering, and payload tests | Enum/decoder + tests | Usually 0 |
| New web-only file or HTTP capability | REST handler + security tests | Web helper | 0 on native |

A simple API should normally take less than half a day. An API involving a new model, permissions, and UI generally takes about one day. More complex business behavior should add effort in the business layer, not through duplicated transport implementations.

### 8.2 Adding an event

Adding an event requires only:

1. Define the event name and payload in Go.
2. Emit it at the relevant business state transition.
3. Add mask-filtering and payload tests.
4. Add a Flutter payload decoder.

The generic FFI callback, gomobile listener, and MethodChannel forwarding must not change for every new event type.

### 8.3 Adding a platform-native capability

File pickers, system notifications, and background location are platform capabilities rather than Gopeed business APIs. Keep them in Flutter plugins or dedicated Platform Channels instead of adding them to the Dispatcher.

## 9. Storage and Hive Removal

### 9.1 Go Storage ownership

The REST server switch, network, address, and token belong in `DownloaderStoreConfig.api`. Window state, creation history, bookmarks, menubar mode, analytics settings, and fallback client ID belong in `DownloaderStoreConfig.extra`. All are written to Go Storage through the unified config API.

Flutter no longer initializes a local database or stores Go connection parameters. Web uses `Uri.base.origin` to access the same-origin REST server that serves the page and does not maintain separate socket, host, or port settings in browser storage. Existing Hive files are neither read nor imported; the Hive package and database wrappers are removed.

### 9.2 Startup sequence

The main window RuntimeController remains the sole owner of the Go Runtime. The app creates the window with safe default dimensions, then restores persisted state after Go is ready:

```text
Initialize the fixed storageDir
→ Create the main window with safe default dimensions
→ Start the Go Runtime
→ Go reads persisted API configuration and starts the optional REST listener when enabled
→ Read Downloader configuration and client preferences
→ Flutter reads desired API configuration and actual REST runtime state separately
→ Restore window dimensions and maximized state
→ Subscribe to terminal task events
```

This preserves the lifecycle rule that the main window owns the Go Runtime and child windows use only Capability RPC. Web authentication uses an HttpOnly Cookie managed by the REST server rather than restoring token storage in Hive.

### 9.3 Legacy data policy

This branch intentionally does not migrate `database.hive`. After upgrade, window state, creation history, and the analytics fallback ID start from defaults. Download tasks, Downloader configuration, and extension data continue to come from existing Go Storage and are unaffected.

### 9.4 Web login state

Web login state must not be stored in `DownloaderStoreConfig.extra`:

- Config is global server state and does not belong to one browser or user.
- The first config request itself requires authentication, so storing the browser token in config creates a bootstrap cycle.
- Config is returned to other authenticated clients, so persisting a browser token there would leak the session.
- Native FFI/gomobile calls do not pass through REST authentication and do not need a web token.

After synchronizing `feat/v2`, the login page and 401 redirect behavior remain in place. On successful login, Go uses `crypto/rand` to create a 256-bit random Session ID. It stores the session and its seven-day expiry in process memory and returns the ID only in a `HttpOnly + SameSite=Lax + Path=/` Cookie. HTTPS and trusted reverse-proxy requests also set `Secure`. The Cookie contains no username, password, or business data, and Flutter Web never reads, persists, or injects the Session ID.

Sessions are not stored in Bolt and do not require AES. Their lifecycle is:

- Browser refresh: the Cookie remains valid and the user stays logged in.
- Browser restart: the session remains valid while the Cookie is unexpired and the Go service has not restarted.
- Go service restart: in-memory sessions are cleared; the old Cookie fails validation, and the first API request returns 401 and opens the login page.
- Seven-day expiry: both browser Cookie and server session expire and require a new login.

The web session protects `/api/**`, `/fs/tasks/**`, and `/fs/extensions/**`. Same-origin API calls, Flutter Web image requests, new-window file access, and downloads automatically include the Cookie. They do not need a separate image authentication header. `X-Api-Token` remains independent and serves CLI and external API clients. If login must survive a Go service restart in the future, add persistent session storage separately instead of mixing it into Downloader business configuration.

A web server cannot be configured with `ApiToken` alone while omitting complete `WebAuth` credentials. Browsers cannot automatically attach `X-Api-Token`, so that configuration would start a server whose web page is unusable. `StartConfig.Validate` rejects it before opening a listener or initializing the Downloader. To serve both the Web UI and external API clients, enable `WebAuth`: the static login page remains public, the browser uses a Session Cookie after login, and external clients use `X-Api-Token`. A native client that does not enable the Web UI may still use `ApiToken` to protect its optional REST server.

## 10. Error Model

Unified error structure:

```json
{
  "code": 1000,
  "message": "task not found",
  "details": {},
  "requestId": "42"
}
```

Error categories:

- Business errors: missing task, invalid arguments, extension installation failure.
- Transport errors: uninitialized FFI, disconnected MethodChannel, REST timeout.
- ABI errors: incompatible versions or invalid return buffers.
- Capability errors: invoking a web-only or native-only capability on the wrong platform.

Flutter converts all of them into `GopeedException`. Components do not handle DioException, PlatformException, or raw FFI errors directly.

## 11. Testing Strategy

### 11.1 Go

- ApplicationService unit tests.
- Dispatcher route and parameter parsing tests.
- Shared request fixtures that verify identical Dispatcher and REST results.
- Runtime repeated-start, shutdown, and independent REST restart tests.
- Tests for desired/runtime state mismatches, native core survival after listener startup failure, and a stopped listener after failed restart.
- Done/error mask filtering, payload, and callback panic-isolation tests.
- Go race tests.
- Repeated FFI buffer allocation/release tests.

### 11.2 Flutter

- Use FakeTransport to test GopeedClient paths, query parameters, request bodies, and decoding.
- RestTransport, DesktopFfiTransport, and MobileGomobileTransport contract tests.
- TaskEvent JSON decoder and subscription-mask tests.
- Desktop notification tests for done/error events.
- Main-window and child-window capability forwarding tests.

### 11.3 Platform smoke tests

- Windows, macOS, Linux: startup, Invoke, events, shutdown, and repeated startup.
- Android, iOS: gomobile Invoke, reverse MethodChannel events, and background/foreground transitions.
- Web: REST, login, proxy, and task-file downloads.
- Native: enabling and disabling REST does not interrupt download tasks.

## 12. Implementation Phases

### Phase 1: Extract the shared backend — implemented on this branch

- Add the shared API Service and explicit Dispatcher.
- Convert existing REST handlers into a Dispatcher adapter.
- Preserve the published core REST contract exactly. New or breaking changes require a new API version and must not modify the existing `/api/v1/**` contract.

### Phase 2: Native in-process calls — implemented on this branch

- Integrate the FFI Transport on Desktop.
- Extend gomobile Invoke and MethodChannel on Mobile.
- Move Flutter API calls to the unified Transport so each typed method is maintained once.
- Do not create a socket listener by default on native clients.

### Phase 3: Terminal task events — implemented on this branch

- Add done/error mask filtering and one callback.
- Add the FFI callback on Desktop and the gomobile listener plus MethodChannel forwarding on Mobile.
- Replace desktop notification polling with events; Web does not implement callbacks.

### Phase 4: Independent REST server lifecycle — implemented on this branch

- Persist REST server configuration.
- Keep REST disabled by default on native clients.
- Continue using the external REST server on Web.
- Expose separate backend operations for state, start, stop, and restart. Lifecycle operations read only persisted configuration.
- Maintain configuration drafts and actual runtime state separately in the settings page. When saving while running, ask whether to save and restart immediately.
- Require confirmation before stopping REST, without closing the Downloader or native in-process API.

### Phase 5: Unified Go Storage — implemented on this branch

- Store Flutter UI preferences in `DownloaderStoreConfig.extra`.
- Restore window state after the Runtime is ready.
- Use the same-origin REST address on Web.
- Explicitly discard legacy Hive data and remove the Hive dependency and database wrappers.

### Phase 6: Cleanup and release

- Remove native socket-based API calls.
- Remove mobile bridge code used only to start REST for native API calls.
- Complete cross-platform contract, memory, lifecycle, event, and stress tests.

## 13. Maintenance Principles

1. Implement business logic once in Go ApplicationService.
2. Register each API route once in the Core Route Registry.
3. Implement each Flutter typed method once in GopeedClient.
4. Transports serialize and forward data; they do not duplicate business methods.
5. Event transports carry a generic envelope and do not expand the native bridge for every event type.
6. Web-only HTTP capabilities do not enter the native API.
7. Flutter widgets do not know whether the transport is REST, FFI, or gomobile.
8. Child windows never own the Go Runtime.

Under these rules, the stable change surface for a normal business API is:

```text
Go Service + Route + Test
Flutter Model (if required) + GopeedClient + Test
```

Maintenance effort does not grow linearly with the number of supported platforms.
