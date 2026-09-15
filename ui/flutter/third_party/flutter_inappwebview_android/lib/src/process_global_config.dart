import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [AndroidProcessGlobalConfig].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformProcessGlobalConfigCreationParams] for
/// more information.
@immutable
class AndroidProcessGlobalConfigCreationParams
    extends PlatformProcessGlobalConfigCreationParams {
  /// Creates a new [AndroidProcessGlobalConfigCreationParams] instance.
  const AndroidProcessGlobalConfigCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformProcessGlobalConfigCreationParams params,
  ) : super();

  /// Creates a [AndroidProcessGlobalConfigCreationParams] instance based on [PlatformProcessGlobalConfigCreationParams].
  factory AndroidProcessGlobalConfigCreationParams.fromPlatformProcessGlobalConfigCreationParams(
      PlatformProcessGlobalConfigCreationParams params) {
    return AndroidProcessGlobalConfigCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformProcessGlobalConfig}
class AndroidProcessGlobalConfig extends PlatformProcessGlobalConfig
    with ChannelController {
  /// Creates a new [AndroidProcessGlobalConfig].
  AndroidProcessGlobalConfig(PlatformProcessGlobalConfigCreationParams params)
      : super.implementation(
          params is AndroidProcessGlobalConfigCreationParams
              ? params
              : AndroidProcessGlobalConfigCreationParams
                  .fromPlatformProcessGlobalConfigCreationParams(params),
        ) {
    channel = const MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_processglobalconfig');
    handler = handleMethod;
    initMethodCallHandler();
  }

  static AndroidProcessGlobalConfig? _instance;

  ///Gets the [AndroidProcessGlobalConfig] shared instance.
  static AndroidProcessGlobalConfig instance() {
    return (_instance != null) ? _instance! : _init();
  }

  static AndroidProcessGlobalConfig _init() {
    _instance = AndroidProcessGlobalConfig(
        AndroidProcessGlobalConfigCreationParams(
            const PlatformProcessGlobalConfigCreationParams()));
    return _instance!;
  }

  Future<dynamic> _handleMethod(MethodCall call) async {}

  @override
  Future<void> apply({required ProcessGlobalConfigSettings settings}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("settings", () => settings.toMap());
    await channel?.invokeMethod('apply', args);
  }

  @override
  void dispose() {
    // empty
  }
}

extension InternalProcessGlobalConfig on AndroidProcessGlobalConfig {
  get handleMethod => _handleMethod;
}
