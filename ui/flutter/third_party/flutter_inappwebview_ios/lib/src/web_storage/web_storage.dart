import 'dart:convert';

import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

import '../in_app_webview/in_app_webview_controller.dart';

/// Object specifying creation parameters for creating a [IOSWebStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebStorageCreationParams] for
/// more information.
class IOSWebStorageCreationParams extends PlatformWebStorageCreationParams {
  /// Creates a new [IOSWebStorageCreationParams] instance.
  IOSWebStorageCreationParams(
      {required super.localStorage, required super.sessionStorage});

  /// Creates a [IOSWebStorageCreationParams] instance based on [PlatformWebStorageCreationParams].
  factory IOSWebStorageCreationParams.fromPlatformWebStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformWebStorageCreationParams params) {
    return IOSWebStorageCreationParams(
        localStorage: params.localStorage,
        sessionStorage: params.sessionStorage);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebStorage}
class IOSWebStorage extends PlatformWebStorage {
  /// Constructs a [IOSWebStorage].
  IOSWebStorage(PlatformWebStorageCreationParams params)
      : super.implementation(
          params is IOSWebStorageCreationParams
              ? params
              : IOSWebStorageCreationParams
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

/// Object specifying creation parameters for creating a [IOSStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformStorageCreationParams] for
/// more information.
class IOSStorageCreationParams extends PlatformStorageCreationParams {
  /// Creates a new [IOSStorageCreationParams] instance.
  IOSStorageCreationParams(
      {required super.controller, required super.webStorageType});

  /// Creates a [IOSStorageCreationParams] instance based on [PlatformStorageCreationParams].
  factory IOSStorageCreationParams.fromPlatformStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformStorageCreationParams params) {
    return IOSStorageCreationParams(
        controller: params.controller, webStorageType: params.webStorageType);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformStorage}
abstract mixin class IOSStorage implements PlatformStorage {
  @override
  IOSInAppWebViewController? controller;

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

/// Object specifying creation parameters for creating a [IOSLocalStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformLocalStorageCreationParams] for
/// more information.
class IOSLocalStorageCreationParams extends PlatformLocalStorageCreationParams {
  /// Creates a new [IOSLocalStorageCreationParams] instance.
  IOSLocalStorageCreationParams(super.params);

  /// Creates a [IOSLocalStorageCreationParams] instance based on [PlatformLocalStorageCreationParams].
  factory IOSLocalStorageCreationParams.fromPlatformLocalStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformLocalStorageCreationParams params) {
    return IOSLocalStorageCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformLocalStorage}
class IOSLocalStorage extends PlatformLocalStorage with IOSStorage {
  /// Constructs a [IOSLocalStorage].
  IOSLocalStorage(PlatformLocalStorageCreationParams params)
      : super.implementation(
          params is IOSLocalStorageCreationParams
              ? params
              : IOSLocalStorageCreationParams
                  .fromPlatformLocalStorageCreationParams(params),
        );

  /// Default storage
  factory IOSLocalStorage.defaultStorage(
      {required PlatformInAppWebViewController? controller}) {
    return IOSLocalStorage(IOSLocalStorageCreationParams(
        PlatformLocalStorageCreationParams(PlatformStorageCreationParams(
            controller: controller,
            webStorageType: WebStorageType.LOCAL_STORAGE))));
  }

  @override
  IOSInAppWebViewController? get controller =>
      params.controller as IOSInAppWebViewController?;
}

/// Object specifying creation parameters for creating a [IOSSessionStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformSessionStorageCreationParams] for
/// more information.
class IOSSessionStorageCreationParams
    extends PlatformSessionStorageCreationParams {
  /// Creates a new [IOSSessionStorageCreationParams] instance.
  IOSSessionStorageCreationParams(super.params);

  /// Creates a [IOSSessionStorageCreationParams] instance based on [PlatformSessionStorageCreationParams].
  factory IOSSessionStorageCreationParams.fromPlatformSessionStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformSessionStorageCreationParams params) {
    return IOSSessionStorageCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformSessionStorage}
class IOSSessionStorage extends PlatformSessionStorage with IOSStorage {
  /// Constructs a [IOSSessionStorage].
  IOSSessionStorage(PlatformSessionStorageCreationParams params)
      : super.implementation(
          params is IOSSessionStorageCreationParams
              ? params
              : IOSSessionStorageCreationParams
                  .fromPlatformSessionStorageCreationParams(params),
        );

  /// Default storage
  factory IOSSessionStorage.defaultStorage(
      {required PlatformInAppWebViewController? controller}) {
    return IOSSessionStorage(IOSSessionStorageCreationParams(
        PlatformSessionStorageCreationParams(PlatformStorageCreationParams(
            controller: controller,
            webStorageType: WebStorageType.SESSION_STORAGE))));
  }

  @override
  IOSInAppWebViewController? get controller =>
      params.controller as IOSInAppWebViewController?;
}
