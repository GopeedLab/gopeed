import 'dart:convert';

import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

import '../in_app_webview/in_app_webview_controller.dart';

/// Object specifying creation parameters for creating a [AndroidWebStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebStorageCreationParams] for
/// more information.
class AndroidWebStorageCreationParams extends PlatformWebStorageCreationParams {
  /// Creates a new [AndroidWebStorageCreationParams] instance.
  AndroidWebStorageCreationParams(
      {required super.localStorage, required super.sessionStorage});

  /// Creates a [AndroidWebStorageCreationParams] instance based on [PlatformWebStorageCreationParams].
  factory AndroidWebStorageCreationParams.fromPlatformWebStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformWebStorageCreationParams params) {
    return AndroidWebStorageCreationParams(
        localStorage: params.localStorage,
        sessionStorage: params.sessionStorage);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebStorage}
class AndroidWebStorage extends PlatformWebStorage {
  /// Constructs a [AndroidWebStorage].
  AndroidWebStorage(PlatformWebStorageCreationParams params)
      : super.implementation(
          params is AndroidWebStorageCreationParams
              ? params
              : AndroidWebStorageCreationParams
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

/// Object specifying creation parameters for creating a [AndroidStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformStorageCreationParams] for
/// more information.
class AndroidStorageCreationParams extends PlatformStorageCreationParams {
  /// Creates a new [AndroidStorageCreationParams] instance.
  AndroidStorageCreationParams(
      {required super.controller, required super.webStorageType});

  /// Creates a [AndroidStorageCreationParams] instance based on [PlatformStorageCreationParams].
  factory AndroidStorageCreationParams.fromPlatformStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformStorageCreationParams params) {
    return AndroidStorageCreationParams(
        controller: params.controller, webStorageType: params.webStorageType);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformStorage}
abstract mixin class AndroidStorage implements PlatformStorage {
  @override
  AndroidInAppWebViewController? controller;

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

/// Object specifying creation parameters for creating a [AndroidLocalStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformLocalStorageCreationParams] for
/// more information.
class AndroidLocalStorageCreationParams
    extends PlatformLocalStorageCreationParams {
  /// Creates a new [AndroidLocalStorageCreationParams] instance.
  AndroidLocalStorageCreationParams(super.params);

  /// Creates a [AndroidLocalStorageCreationParams] instance based on [PlatformLocalStorageCreationParams].
  factory AndroidLocalStorageCreationParams.fromPlatformLocalStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformLocalStorageCreationParams params) {
    return AndroidLocalStorageCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformLocalStorage}
class AndroidLocalStorage extends PlatformLocalStorage with AndroidStorage {
  /// Constructs a [AndroidLocalStorage].
  AndroidLocalStorage(PlatformLocalStorageCreationParams params)
      : super.implementation(
          params is AndroidLocalStorageCreationParams
              ? params
              : AndroidLocalStorageCreationParams
                  .fromPlatformLocalStorageCreationParams(params),
        );

  /// Default storage
  factory AndroidLocalStorage.defaultStorage(
      {required PlatformInAppWebViewController? controller}) {
    return AndroidLocalStorage(AndroidLocalStorageCreationParams(
        PlatformLocalStorageCreationParams(PlatformStorageCreationParams(
            controller: controller,
            webStorageType: WebStorageType.LOCAL_STORAGE))));
  }

  @override
  AndroidInAppWebViewController? get controller =>
      params.controller as AndroidInAppWebViewController?;
}

/// Object specifying creation parameters for creating a [AndroidSessionStorage].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformSessionStorageCreationParams] for
/// more information.
class AndroidSessionStorageCreationParams
    extends PlatformSessionStorageCreationParams {
  /// Creates a new [AndroidSessionStorageCreationParams] instance.
  AndroidSessionStorageCreationParams(super.params);

  /// Creates a [AndroidSessionStorageCreationParams] instance based on [PlatformSessionStorageCreationParams].
  factory AndroidSessionStorageCreationParams.fromPlatformSessionStorageCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformSessionStorageCreationParams params) {
    return AndroidSessionStorageCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformSessionStorage}
class AndroidSessionStorage extends PlatformSessionStorage with AndroidStorage {
  /// Constructs a [AndroidSessionStorage].
  AndroidSessionStorage(PlatformSessionStorageCreationParams params)
      : super.implementation(
          params is AndroidSessionStorageCreationParams
              ? params
              : AndroidSessionStorageCreationParams
                  .fromPlatformSessionStorageCreationParams(params),
        );

  /// Default storage
  factory AndroidSessionStorage.defaultStorage(
      {required PlatformInAppWebViewController? controller}) {
    return AndroidSessionStorage(AndroidSessionStorageCreationParams(
        PlatformSessionStorageCreationParams(PlatformStorageCreationParams(
            controller: controller,
            webStorageType: WebStorageType.SESSION_STORAGE))));
  }

  @override
  AndroidInAppWebViewController? get controller =>
      params.controller as AndroidInAppWebViewController?;
}
