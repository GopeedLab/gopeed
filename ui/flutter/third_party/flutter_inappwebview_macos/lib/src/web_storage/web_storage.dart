import 'dart:convert';

import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

import '../in_app_webview/in_app_webview_controller.dart';

/// Object specifying creation parameters for creating a [MacOSWebStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebStorageCreationParams] for
/// more information.
class MacOSWebStorageCreationParams extends PlatformWebStorageCreationParams {
  /// Creates a new [MacOSWebStorageCreationParams] instance.
  MacOSWebStorageCreationParams(
      {required super.localStorage, required super.sessionStorage});

  /// Creates a [MacOSWebStorageCreationParams] instance based on [PlatformWebStorageCreationParams].
  factory MacOSWebStorageCreationParams.fromPlatformWebStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformWebStorageCreationParams params) {
    return MacOSWebStorageCreationParams(
        localStorage: params.localStorage,
        sessionStorage: params.sessionStorage);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebStorage}
class MacOSWebStorage extends PlatformWebStorage {
  /// Constructs a [MacOSWebStorage].
  MacOSWebStorage(PlatformWebStorageCreationParams params)
      : super.implementation(
          params is MacOSWebStorageCreationParams
              ? params
              : MacOSWebStorageCreationParams
                  .fromPlatformWebStorageCreationParams(params),
        );

  @override
  PlatformLocalStorage get localStorage => params.localStorage;

  @override
  PlatformSessionStorage get sessionStorage => params.sessionStorage;

  @override
  void dispose() {
    localStorage.dispose();
    sessionStorage.dispose();
  }
}

/// Object specifying creation parameters for creating a [MacOSStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformStorageCreationParams] for
/// more information.
class MacOSStorageCreationParams extends PlatformStorageCreationParams {
  /// Creates a new [MacOSStorageCreationParams] instance.
  MacOSStorageCreationParams(
      {required super.controller, required super.webStorageType});

  /// Creates a [MacOSStorageCreationParams] instance based on [PlatformStorageCreationParams].
  factory MacOSStorageCreationParams.fromPlatformStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformStorageCreationParams params) {
    return MacOSStorageCreationParams(
        controller: params.controller, webStorageType: params.webStorageType);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformStorage}
abstract mixin class MacOSStorage implements PlatformStorage {
  @override
  MacOSInAppWebViewController? controller;

  @override
  Future<int?> length() async {
    var result = await controller?.evaluateJavascript(source: """
    window.$webStorageType.length;
    """);
    return result != null ? int.parse(json.decode(result)) : null;
  }

  @override
  Future<void> setItem({required String key, required dynamic value}) async {
    var encodedValue = json.encode(value);
    await controller?.evaluateJavascript(source: """
    window.$webStorageType.setItem("$key", ${value is String ? encodedValue : "JSON.stringify($encodedValue)"});
    """);
  }

  @override
  Future<dynamic> getItem({required String key}) async {
    var itemValue = await controller?.evaluateJavascript(source: """
    window.$webStorageType.getItem("$key");
    """);

    if (itemValue == null) {
      return null;
    }

    try {
      return json.decode(itemValue);
    } catch (e) {}

    return itemValue;
  }

  @override
  Future<void> removeItem({required String key}) async {
    await controller?.evaluateJavascript(source: """
    window.$webStorageType.removeItem("$key");
    """);
  }

  @override
  Future<List<WebStorageItem>> getItems() async {
    var webStorageItems = <WebStorageItem>[];

    List<Map<dynamic, dynamic>>? items =
        (await controller?.evaluateJavascript(source: """
(function() {
  var webStorageItems = [];
  for(var i = 0; i < window.$webStorageType.length; i++){
    var key = window.$webStorageType.key(i);
    webStorageItems.push(
      {
        key: key,
        value: window.$webStorageType.getItem(key)
      }
    );
  }
  return webStorageItems;
})();
    """))?.cast<Map<dynamic, dynamic>>();

    if (items == null) {
      return webStorageItems;
    }

    for (var item in items) {
      webStorageItems
          .add(WebStorageItem(key: item["key"], value: item["value"]));
    }

    return webStorageItems;
  }

  @override
  Future<void> clear() async {
    await controller?.evaluateJavascript(source: """
    window.$webStorageType.clear();
    """);
  }

  @override
  Future<String> key({required int index}) async {
    var result = await controller?.evaluateJavascript(source: """
    window.$webStorageType.key($index);
    """);
    return result != null ? json.decode(result) : null;
  }

  @override
  void dispose() {
    controller = null;
  }
}

/// Object specifying creation parameters for creating a [MacOSLocalStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformLocalStorageCreationParams] for
/// more information.
class MacOSLocalStorageCreationParams
    extends PlatformLocalStorageCreationParams {
  /// Creates a new [MacOSLocalStorageCreationParams] instance.
  MacOSLocalStorageCreationParams(super.params);

  /// Creates a [MacOSLocalStorageCreationParams] instance based on [PlatformLocalStorageCreationParams].
  factory MacOSLocalStorageCreationParams.fromPlatformLocalStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformLocalStorageCreationParams params) {
    return MacOSLocalStorageCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformLocalStorage}
class MacOSLocalStorage extends PlatformLocalStorage with MacOSStorage {
  /// Constructs a [MacOSLocalStorage].
  MacOSLocalStorage(PlatformLocalStorageCreationParams params)
      : super.implementation(
          params is MacOSLocalStorageCreationParams
              ? params
              : MacOSLocalStorageCreationParams
                  .fromPlatformLocalStorageCreationParams(params),
        );

  /// Default storage
  factory MacOSLocalStorage.defaultStorage(
      {required PlatformInAppWebViewController? controller}) {
    return MacOSLocalStorage(MacOSLocalStorageCreationParams(
        PlatformLocalStorageCreationParams(PlatformStorageCreationParams(
            controller: controller,
            webStorageType: WebStorageType.LOCAL_STORAGE))));
  }

  @override
  MacOSInAppWebViewController? get controller =>
      params.controller as MacOSInAppWebViewController?;
}

/// Object specifying creation parameters for creating a [MacOSSessionStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformSessionStorageCreationParams] for
/// more information.
class MacOSSessionStorageCreationParams
    extends PlatformSessionStorageCreationParams {
  /// Creates a new [MacOSSessionStorageCreationParams] instance.
  MacOSSessionStorageCreationParams(super.params);

  /// Creates a [MacOSSessionStorageCreationParams] instance based on [PlatformSessionStorageCreationParams].
  factory MacOSSessionStorageCreationParams.fromPlatformSessionStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformSessionStorageCreationParams params) {
    return MacOSSessionStorageCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformSessionStorage}
class MacOSSessionStorage extends PlatformSessionStorage with MacOSStorage {
  /// Constructs a [MacOSSessionStorage].
  MacOSSessionStorage(PlatformSessionStorageCreationParams params)
      : super.implementation(
          params is MacOSSessionStorageCreationParams
              ? params
              : MacOSSessionStorageCreationParams
                  .fromPlatformSessionStorageCreationParams(params),
        );

  /// Default storage
  factory MacOSSessionStorage.defaultStorage(
      {required PlatformInAppWebViewController? controller}) {
    return MacOSSessionStorage(MacOSSessionStorageCreationParams(
        PlatformSessionStorageCreationParams(PlatformStorageCreationParams(
            controller: controller,
            webStorageType: WebStorageType.SESSION_STORAGE))));
  }

  @override
  MacOSInAppWebViewController? get controller =>
      params.controller as MacOSInAppWebViewController?;
}
