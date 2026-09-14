# Native WebView host bridge

Vendored unmodified Dart facades and native sources from pub.dev:

- flutter_inappwebview_macos 1.1.2
- flutter_inappwebview_ios 1.1.2
- flutter_inappwebview_android 1.1.3

Each package retains its upstream license. Versions and original archive hashes
are available in the parent commit's pubspec.lock.

Local native changes are limited to:

- `GopeedProfiles`: remove profiles on uninstall and prepare a persistent profile and apply the internal loopback
  proxy before creating a WebView. Apple uses SOCKS5 to cover HTTP as well as HTTPS;
  Android uses the application-wide HTTP proxy override completion callback.
- `InAppWebViewSettings` and `InAppWebView`: read the host-only `gopeedProfileId`
  setting and attach the profile before navigation.
- `MyCookieManager`: route cookie operations to the requested profile, capturing
  the store per operation so asynchronous calls cannot cross profiles.

`lib/app/rpc/webview_profile.dart` uses the existing plugin method channel and
serializes these internal settings. No extension JavaScript API is added.

Apple requires macOS 14 / iOS 17 for named persistent stores and proxy settings.
Android requires `MULTI_PROFILE` and `PROXY_OVERRIDE`; unsupported runtimes return
an explicit unavailable error rather than opening a shared, unproxied browser.

When updating upstream, refresh these three packages, reapply the above changes,
and run `tool/webview_profile_check.dart` on a native Apple host. Do not patch the
user's pub cache: dependency_overrides selects these tracked sources reproducibly.

Android uses AndroidX WebKit 1.13.0 for profile-scoped full browsing-data deletion.
Loaded profile shells are queued durably for deletion before WebViews initialize
at the next process start. No global cookie/cache clearing is used for uninstall.
