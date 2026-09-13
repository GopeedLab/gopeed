import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [AndroidServiceWorkerController].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformServiceWorkerControllerCreationParams] for
/// more information.
@immutable
class AndroidServiceWorkerControllerCreationParams
    extends PlatformServiceWorkerControllerCreationParams {
  /// Creates a new [AndroidServiceWorkerControllerCreationParams] instance.
  const AndroidServiceWorkerControllerCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformServiceWorkerControllerCreationParams params,
  ) : super();

  /// Creates a [AndroidServiceWorkerControllerCreationParams] instance based on [PlatformServiceWorkerControllerCreationParams].
  factory AndroidServiceWorkerControllerCreationParams.fromPlatformServiceWorkerControllerCreationParams(
      PlatformServiceWorkerControllerCreationParams params) {
    return AndroidServiceWorkerControllerCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformServiceWorkerController}
class AndroidServiceWorkerController extends PlatformServiceWorkerController
    with ChannelController {
  /// Creates a new [AndroidServiceWorkerController].
  AndroidServiceWorkerController(
      PlatformServiceWorkerControllerCreationParams params)
      : super.implementation(
          params is AndroidServiceWorkerControllerCreationParams
              ? params
              : AndroidServiceWorkerControllerCreationParams
                  .fromPlatformServiceWorkerControllerCreationParams(params),
        ) {
    channel = const MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_serviceworkercontroller');
    handler = handleMethod;
    initMethodCallHandler();
  }

  factory AndroidServiceWorkerController.static() {
    return instance();
  }

  static AndroidServiceWorkerController? _instance;

  ///Gets the [AndroidServiceWorkerController] shared instance.
  static AndroidServiceWorkerController instance() {
    return (_instance != null) ? _instance! : _init();
  }

  static AndroidServiceWorkerController _init() {
    _instance = AndroidServiceWorkerController(
        AndroidServiceWorkerControllerCreationParams(
            const PlatformServiceWorkerControllerCreationParams()));
    return _instance!;
  }

  ServiceWorkerClient? _serviceWorkerClient;

  @override
  ServiceWorkerClient? get serviceWorkerClient => _serviceWorkerClient;

  @override
  Future<void> setServiceWorkerClient(ServiceWorkerClient? value) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('isNull', () => value == null);
    await channel?.invokeMethod("setServiceWorkerClient", args);
    _serviceWorkerClient = value;
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    switch (call.method) {
      case "shouldInterceptRequest":
        if (serviceWorkerClient != null &&
            serviceWorkerClient!.shouldInterceptRequest != null) {
          Map<String, dynamic> arguments =
              call.arguments.cast<String, dynamic>();
          WebResourceRequest request = WebResourceRequest.fromMap(arguments)!;

          return (await serviceWorkerClient!.shouldInterceptRequest!(request))
              ?.toMap();
        }
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }

    return null;
  }

  @override
  Future<bool> getAllowContentAccess() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('getAllowContentAccess', args) ??
        false;
  }

  @override
  Future<bool> getAllowFileAccess() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('getAllowFileAccess', args) ??
        false;
  }

  @override
  Future<bool> getBlockNetworkLoads() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('getBlockNetworkLoads', args) ??
        false;
  }

  @override
  Future<CacheMode?> getCacheMode() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return CacheMode.fromNativeValue(
        await channel?.invokeMethod<int?>('getCacheMode', args));
  }

  @override
  Future<void> setAllowContentAccess(bool allow) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("allow", () => allow);
    await channel?.invokeMethod('setAllowContentAccess', args);
  }

  @override
  Future<void> setAllowFileAccess(bool allow) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("allow", () => allow);
    await channel?.invokeMethod('setAllowFileAccess', args);
  }

  @override
  Future<void> setBlockNetworkLoads(bool flag) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("flag", () => flag);
    await channel?.invokeMethod('setBlockNetworkLoads', args);
  }

  @override
  Future<void> setCacheMode(CacheMode mode) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("mode", () => mode.toNativeValue());
    await channel?.invokeMethod('setCacheMode', args);
  }

  @override
  void dispose() {
    // empty
  }
}

extension InternalServiceWorkerController on AndroidServiceWorkerController {
  get handleMethod => _handleMethod;
}
