# Gopeed Flutter client

The Gopeed 2.0 client runs on Windows, macOS, Linux, Android, iOS, and the web. Desktop uses Dart FFI to call the Go core. Android and iOS use platform channels and gomobile. The web client uses the Go REST API.

Use Go 1.27.4 and Flutter 3.41.6 (Dart 3.11.4). The [contributing guide](../../CONTRIBUTING.md) covers local web development, code generation, and tests. The [project README](../../README.md) has the platform build commands.

See the [backend transport architecture](docs/backend-transport-architecture.md) and [desktop window capability contract](docs/window-capability-rpc.md) before changing how the client calls Go or how desktop windows share state.
