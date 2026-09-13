import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [AndroidWebViewFeature].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebViewFeatureCreationParams] for
/// more information.
@immutable
class AndroidWebViewFeatureCreationParams
    extends PlatformWebViewFeatureCreationParams {
  /// Creates a new [AndroidWebViewFeatureCreationParams] instance.
  const AndroidWebViewFeatureCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformWebViewFeatureCreationParams params,
  ) : super();

  /// Creates a [AndroidWebViewFeatureCreationParams] instance based on [PlatformWebViewFeatureCreationParams].
  factory AndroidWebViewFeatureCreationParams.fromPlatformWebViewFeatureCreationParams(
      PlatformWebViewFeatureCreationParams params) {
    return AndroidWebViewFeatureCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebViewFeature}
class AndroidWebViewFeature extends PlatformWebViewFeature
    with ChannelController {
  /// Creates a new [AndroidWebViewFeature].
  AndroidWebViewFeature(PlatformWebViewFeatureCreationParams params)
      : super.implementation(
          params is AndroidWebViewFeatureCreationParams
              ? params
              : AndroidWebViewFeatureCreationParams
                  .fromPlatformWebViewFeatureCreationParams(params),
        ) {
    channel = const MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_webviewfeature');
    handler = handleMethod;
    initMethodCallHandler();
  }

  factory AndroidWebViewFeature.static() {
    return instance();
  }

  static AndroidWebViewFeature? _instance;

  ///Gets the [AndroidWebViewFeature] shared instance.
  static AndroidWebViewFeature instance() {
    return (_instance != null) ? _instance! : _init();
  }

  static AndroidWebViewFeature _init() {
    _instance = AndroidWebViewFeature(AndroidWebViewFeatureCreationParams(
        const PlatformWebViewFeatureCreationParams()));
    return _instance!;
  }

  Future<dynamic> _handleMethod(MethodCall call) async {}

  @override
  Future<bool> isFeatureSupported(WebViewFeature feature) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("feature", () => feature.toNativeValue());
    return await channel?.invokeMethod<bool>('isFeatureSupported', args) ??
        false;
  }

  @override
  Future<bool> isStartupFeatureSupported(WebViewFeature startupFeature) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("startupFeature", () => startupFeature.toNativeValue());
    return await channel?.invokeMethod<bool>(
            'isStartupFeatureSupported', args) ??
        false;
  }

  @override
  void dispose() {
    // empty
  }
}

extension InternalWebViewFeature on AndroidWebViewFeature {
  get handleMethod => _handleMethod;
}
