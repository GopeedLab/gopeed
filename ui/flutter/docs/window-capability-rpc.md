# Window Capability RPC Architecture

This document defines the internal communication contract between the main window and desktop child windows.

## 1. Ownership

- The main window is the only owner of the Gopeed runtime, Gopeed API connection, and Hive database.
- Child windows must never initialize `libgopeed`, initialize the Gopeed HTTP client, or open a Hive box.
- Child windows access application capabilities through `AppCapabilities` only.
- Immutable window launch input, such as the window type and an initial create-task request, may use `AppWindowPayload`.
- API addresses, API tokens, appearance state, locale, and persisted data must not be placed in launch payloads.

## 2. Layers

```text
Presentation / Riverpod controllers
              |
        AppCapabilities
          /         \
 GopeedService   AppStorageService
          \         /
       CapabilityInvoker
          /         \
 LocalCapability   WindowCapability
     Invoker           Invoker
          |               |
 CapabilityRegistry  WindowMethodChannel
          |               |
 Gopeed HTTP/FFI + Hive in the main window
```

The presentation layer must not branch on whether it is running in a main or child window. The root `ProviderScope` selects the local or remote capability implementation.

## 3. RPC Contract

Every operation is declared once as a typed `RpcMethod<P, R>`:

```dart
static const resolve = RpcMethod<ResolveTask, ResolveResult>('gopeed.resolve');
```

Rules:

- Operation names are stable protocol identifiers and must be declared in a capability method catalog.
- Do not duplicate operation strings in pages, controllers, host handlers, or clients.
- Do not create one MethodChannel per operation.
- Do not expose Gopeed HTTP paths through the internal capability API.
- Multi-argument operations should use a JSON map or a dedicated request DTO.
- Prefer a dedicated request DTO when parameters have domain meaning or are likely to evolve.

`RpcCodecRegistry` owns serialization. Encoding is generic for JSON primitives, collections, enums, and models with `toJson()`. Each non-primitive response or request model registers its `fromJson` decoder once. Individual operations must not repeat request and response codec closures.

The local invoker calls the typed handler directly and does not perform a JSON round trip. Serialization occurs only at a window boundary.

## 4. Transport

Child-to-main requests use one unidirectional `WindowMethodChannel`:

```text
gopeed.app.capabilities.v1
```

The transport uses one method, `capability.call`, with an operation name and JSON payload. Results use a common success/error envelope. MethodChannel provides request-response correlation, so the protocol does not add request IDs for ordinary calls.

The existing HTTP `HostRpcService` is an external browser-extension integration and is not the internal window capability transport. Its `/forward` endpoint must not be reused by child windows.

## 5. Events And State Synchronization

Main-to-child events use each child's `WindowController` channel. A child registers its method handler before subscribing.

Initialization order:

1. The child registers its window event handler.
2. The child requests the current appearance snapshot.
3. The child applies the snapshot.
4. The child subscribes with its window ID.
5. The main window stores the controller and immediately sends the current snapshot again.
6. Later state changes are broadcast to every subscribed child.

Events contain complete state snapshots, not patches. No revision field is required. Event names are declared centrally in `AppWindowRpcProtocol`.

Appearance synchronization uses `appearance.changed` and currently contains:

- Theme mode
- Theme color
- Locale

State broadcasting must observe the owning Riverpod state centrally. Do not broadcast directly from a settings button callback, because updates can originate from configuration loading, system changes, or future entry points.

## 6. Storage

- `Database` remains the low-level Hive implementation and is main-window infrastructure.
- Product code used by child windows depends on `AppStorageService`.
- Storage RPC methods expose business capabilities such as create-history operations, not raw `box.get` or `box.put` access.
- Prefer batch mutations across the window boundary. For example, save all parsed URLs in one `saveCreateHistory(List<String>)` call.
- New child-window storage requirements must be added to `StorageMethods` and bound in the main registry.
- Config reads, full config saves, and create-history mutations share one main-window queue. Each history mutation reads, changes, and writes the config within that queue. History reads wait for earlier saves.
- `StorageMethods` owns `extra.createHistory`; full config saves preserve its latest value inside the queue so stale settings or child-window snapshots cannot overwrite history.

## 7. Gopeed Backend Migration

All structured task, configuration, extension, and webhook operations are exposed through `GopeedService`. The local registry currently binds them to `lib/api/api.dart`.

When the backend moves to FFI:

- Replace the main-window local bindings or their underlying implementation.
- Keep `GopeedMethods`, `GopeedService`, child-window transport, pages, and controllers unchanged.
- Keep raw HTTP proxy functionality separate unless an explicit capability contract is introduced for it.

## 8. Adding A Capability

1. Add one typed `RpcMethod` to the appropriate method catalog.
2. Register a model decoder once if a new non-primitive type crosses the boundary.
3. Add the strongly typed facade method to the capability service.
4. Bind the method to the main-window implementation in `LocalAppCapabilities`.
5. Use the facade from a Riverpod controller or page.
6. Add tests for local typed dispatch and serialized dispatch.
7. For events, add one centralized event name and broadcast a complete snapshot.

## 9. Prohibited Patterns

Child-window code must not:

- Import `lib/api/api.dart`.
- Call `api.init` or access the Gopeed socket/address/token.
- Import or access `Database.instance`.
- Open Hive boxes.
- Receive API configuration or persisted data through `AppWindowPayload`.
- Add feature-specific MethodChannels.
- Parse RPC JSON directly inside a page or widget.

These constraints keep all backend and persistence ownership in the main window and allow multiple child windows to operate concurrently without database locks or duplicated Gopeed runtimes.

## 10. Batch Task Creation

`GopeedMethods.createTaskBatch` carries `CreateTaskBatch` through the existing
codec registry. Each member accepts either `rid` from resolve or `req`, plus
creation `opts`. A nonempty `rid` takes priority. BT selections retain their
original file indexes in `opts.selectFiles`.

Both ordinary and grouped creation return `CreateTaskBatchResult`, an object
with `taskIds` and an optional `parentId`. A successful response completes the
batch, including partial success. The preview tracks completed batches so a
failed batch can be retried without repeating successful batches. Creation does
not accept a group ID to append members to an existing group.

## 11. Opening Child Surfaces

Use `AppRouter.openWindow(AppWindowPayload)` for child-window destinations.
`AppRouter.openCreateTask(createTask: ...)` is the create-task convenience API.
All entry points, including deep links, browser capture and the tray, use this
API rather than selecting a presentation themselves:

- Desktop launches an independent native child window through
  `AppWindowLauncher.open(payload)` and the existing capability transport.
- Web pushes an `AppWindowPage` on the root Navigator. This routed shadcn modal
  keeps the current section, filters and scroll state mounted underneath it.
- Native mobile retains the full-page flow, including later share requests.
- If native child-window creation fails, the main window is brought forward
  and the same in-app modal is used as a fallback.

`AppWindowContent` owns the destination paths and content builders shared with
`ChildWindowApp`. For another child surface, add its payload type, register its
path and content, and use `AppWindowPage` for its in-app route. The modal's
preferred size can be supplied per destination. Keep backend and storage access
on the existing capability API in either presentation.

Create-task content reports cancellation or completion through `onClose` when
hosted in-app. Cancellation pops only the child surface. Completion selects the
downloading list and returns home. The route payload uses a serializable string
so browser history can restore a modal and its initial request.

Download progress uses `AppWindowPayload.downloadProgress(taskId)` through the
same destination registry, native window launcher and routed fallback. Its
payload contains only the task ID. The page reuses `TasksController` and
`TaskStatisticsTab`; polling, pause/resume and confirmed cancellation use the
existing Gopeed capabilities in either window. Closing or backgrounding the
window does not change the task. Completed tasks expose native file actions.
The task context menu can open or reopen a status window independently of the
automatic-opening setting, including for multiple different tasks.

Each status window observes successful task snapshots to collect a timestamped
speed history, beginning when the window opens. The full observed time range
fills the chart width and compresses as samples arrive. Completion freezes the
history; subsequent polling never appends zero-speed samples. API outages leave
gaps, and paused/waiting/completed tasks report
zero instantaneous download speed. Average speed uses downloaded bytes and
backend active download duration, not the chart samples. No additional backend
poller, simulated data, or child-owned runtime is introduced.

The main shell owns `DownloadProgressWindowController`. On desktop it observes
successful task-list snapshots and opens one window per new top-level task
(one per group). The first snapshot establishes a baseline; existing tasks and
subsequent status changes do not open windows. The persisted
`extra.showDownloadProgressWindow` preference defaults to false and controls
future automatic openings without closing windows already in use.
