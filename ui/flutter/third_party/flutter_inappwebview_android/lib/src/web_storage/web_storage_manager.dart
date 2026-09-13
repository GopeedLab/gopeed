import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [AndroidWebStorageManager].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebStorageManagerCreationParams] for
/// more information.
@immutable
class AndroidWebStorageManagerCreationParams
    extends PlatformWebStorageManagerCreationParams {
  /// Creates a new [AndroidWebStorageManagerCreationParams] instance.
  const AndroidWebStorageManagerCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformWebStorageManagerCreationParams params,
  ) : super();

  /// Creates a [AndroidWebStorageManagerCreationParams] instance based on [PlatformWebStorageManagerCreationParams].
  factory AndroidWebStorageManagerCreationParams.fromPlatformWebStorageManagerCreationParams(
      PlatformWebStorageManagerCreationParams params) {
    return AndroidWebStorageManagerCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebStorageManager}
class AndroidWebStorageManager extends PlatformWebStorageManager
    with ChannelController {
  /// Creates a new [AndroidWebStorageManager].
  AndroidWebStorageManager(PlatformWebStorageManagerCreationParams params)
      : super.implementation(
          params is AndroidWebStorageManagerCreationParams
              ? params
              : AndroidWebStorageManagerCreationParams
                  .fromPlatformWebStorageManagerCreationParams(params),
        ) {
    channel = const MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_webstoragemanager');
    handler = handleMethod;
    initMethodCallHandler();
  }

  static AndroidWebStorageManager? _instance;

  ///Gets the WebStorage manager shared instance.
  static AndroidWebStorageManager instance() {
    return (_instance != null) ? _instance! : _init();
  }

  static AndroidWebStorageManager _init() {
    _instance = AndroidWebStorageManager(AndroidWebStorageManagerCreationParams(
        const PlatformWebStorageManagerCreationParams()));
    return _instance!;
  }

  Future<dynamic> _handleMethod(MethodCall call) async {}

  @override
  Future<List<WebStorageOrigin>> getOrigins() async {
    List<WebStorageOrigin> originsList = [];

    Map<String, dynamic> args = <String, dynamic>{};
    List<Map<dynamic, dynamic>> origins =
        (await channel?.invokeMethod<List>('getOrigins', args))
                ?.cast<Map<dynamic, dynamic>>() ??
            [];

    for (var origin in origins) {
      originsList.add(WebStorageOrigin(
          origin: origin["origin"],
          quota: origin["quota"],
          usage: origin["usage"]));
    }

    return originsList;
  }

  @override
  Future<void> deleteAllData() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('deleteAllData', args);
  }

  @override
  Future<void> deleteOrigin({required String origin}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("origin", () => origin);
    await channel?.invokeMethod('deleteOrigin', args);
  }

  @override
  Future<int> getQuotaForOrigin({required String origin}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("origin", () => origin);
    return await channel?.invokeMethod<int>('getQuotaForOrigin', args) ?? 0;
  }

  @override
  Future<int> getUsageForOrigin({required String origin}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("origin", () => origin);
    return await channel?.invokeMethod<int>('getUsageForOrigin', args) ?? 0;
  }

  @override
  void dispose() {
    // empty
  }
}

extension InternalWebStorageManager on AndroidWebStorageManager {
  get handleMethod => _handleMethod;
}
