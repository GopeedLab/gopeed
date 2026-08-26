import 'browser_extension_host_stub.dart'
    if (dart.library.io) 'entry/browser_extension_host_native.dart';

enum Browser { chrome, edge, firefox }

/// Install host binary for browser extension
Future<void> installHost() => doInstallHost();

/// Check if specified browser is installed
Future<bool> checkBrowserInstalled(Browser browser) =>
    doCheckBrowserInstalled(browser);

/// Check if browser extension manifest is properly installed
Future<bool> checkManifestInstalled(Browser browser) =>
    doCheckManifestInstalled(browser);

/// Install browser extension manifest
Future<void> installManifest(Browser browser) => doInstallManifest(browser);
