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
