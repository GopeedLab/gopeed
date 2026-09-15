import 'dart:ui';

import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [IOSPullToRefreshController].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformPullToRefreshControllerCreationParams] for
/// more information.
class IOSPullToRefreshControllerCreationParams
    extends PlatformPullToRefreshControllerCreationParams {
  /// Creates a new [IOSPullToRefreshControllerCreationParams] instance.
  IOSPullToRefreshControllerCreationParams(
      {super.onRefresh, super.options, super.settings});

  /// Creates a [IOSPullToRefreshControllerCreationParams] instance based on [PlatformPullToRefreshControllerCreationParams].
  factory IOSPullToRefreshControllerCreationParams.fromPlatformPullToRefreshControllerCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformPullToRefreshControllerCreationParams params) {
    return IOSPullToRefreshControllerCreationParams(
        onRefresh: params.onRefresh,
        options: params.options,
        settings: params.settings);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformPullToRefreshController}
class IOSPullToRefreshController extends PlatformPullToRefreshController
    with ChannelController {
  /// Constructs a [IOSPullToRefreshController].
  IOSPullToRefreshController(
      PlatformPullToRefreshControllerCreationParams params)
      : super.implementation(
          params is IOSPullToRefreshControllerCreationParams
              ? params
              : IOSPullToRefreshControllerCreationParams
                  .fromPlatformPullToRefreshControllerCreationParams(params),
        );

  _debugLog(String method, dynamic args) {
    debugLog(
        className: this.runtimeType.toString(),
        debugLoggingSettings:
            PlatformPullToRefreshController.debugLoggingSettings,
        method: method,
        args: args);
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    _debugLog(call.method, call.arguments);

    switch (call.method) {
      case "onRefresh":
        if (params.onRefresh != null) params.onRefresh!();
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
    return null;
  }

  @override
  Future<void> setEnabled(bool enabled) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('enabled', () => enabled);
    await channel?.invokeMethod('setEnabled', args);
  }

  @override
  Future<bool> isEnabled() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('isEnabled', args) ?? false;
  }

  Future<void> _setRefreshing(bool refreshing) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('refreshing', () => refreshing);
    await channel?.invokeMethod('setRefreshing', args);
  }

  @override
  Future<void> beginRefreshing() async {
    return await _setRefreshing(true);
  }

  @override
  Future<void> endRefreshing() async {
    await _setRefreshing(false);
  }

  @override
  Future<bool> isRefreshing() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('isRefreshing', args) ?? false;
  }

  @override
  Future<void> setColor(Color color) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('color', () => color.toHex());
    await channel?.invokeMethod('setColor', args);
  }

  @override
  Future<void> setBackgroundColor(Color color) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('color', () => color.toHex());
    await channel?.invokeMethod('setBackgroundColor', args);
  }

  @Deprecated("Use setStyledTitle instead")
  @override
  Future<void> setAttributedTitle(IOSNSAttributedString attributedTitle) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('attributedTitle', () => attributedTitle.toMap());
    await channel?.invokeMethod('setStyledTitle', args);
  }

  @override
  Future<void> setStyledTitle(AttributedString attributedTitle) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('attributedTitle', () => attributedTitle.toMap());
    await channel?.invokeMethod('setStyledTitle', args);
  }

  @override
  void dispose({bool isKeepAlive = false}) {
    disposeChannel(removeMethodCallHandler: !isKeepAlive);
  }
}

extension InternalPullToRefreshController on IOSPullToRefreshController {
  void init(dynamic id) {
    channel = MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_pull_to_refresh_$id');
    handler = _handleMethod;
    initMethodCallHandler();
  }
}
