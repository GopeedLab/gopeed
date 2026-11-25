# Gopeed Flutter client

The Gopeed 2.0 client runs on Windows, macOS, Linux, Android, iOS, and the web. Desktop uses Dart FFI to call the Go core. Android and iOS use platform channels and gomobile. The web client uses the Go REST API.

Use Go 1.27.4 and Flutter 3.41.6 (Dart 3.11.4). See the [project README](../../README.md) for platform build commands and the [translation guide](../../CONTRIBUTING.md#translation) for localization contributions.

See the [backend transport architecture](docs/backend-transport-architecture.md) and [desktop window capability contract](docs/window-capability-rpc.md) before changing how the client calls Go or how desktop windows share state.
