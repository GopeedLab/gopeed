import 'dart:async';

import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [MacOSWebAuthenticationSession].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebAuthenticationSessionCreationParams] for
/// more information.
class MacOSWebAuthenticationSessionCreationParams
    extends PlatformWebAuthenticationSessionCreationParams {
  /// Creates a new [MacOSWebAuthenticationSessionCreationParams] instance.
  const MacOSWebAuthenticationSessionCreationParams();

  /// Creates a [MacOSWebAuthenticationSessionCreationParams] instance based on [PlatformWebAuthenticationSessionCreationParams].
  factory MacOSWebAuthenticationSessionCreationParams.fromPlatformWebAuthenticationSessionCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformWebAuthenticationSessionCreationParams params) {
    return MacOSWebAuthenticationSessionCreationParams();
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebAuthenticationSession}
class MacOSWebAuthenticationSession extends PlatformWebAuthenticationSession
    with ChannelController {
  /// Constructs a [MacOSWebAuthenticationSession].
  MacOSWebAuthenticationSession(
      PlatformWebAuthenticationSessionCreationParams params)
      : super.implementation(
          params is MacOSWebAuthenticationSessionCreationParams
              ? params
              : MacOSWebAuthenticationSessionCreationParams
                  .fromPlatformWebAuthenticationSessionCreationParams(params),
        );

  static final MacOSWebAuthenticationSession _staticValue =
      MacOSWebAuthenticationSession(
          MacOSWebAuthenticationSessionCreationParams());

  /// Provide static access.
  factory MacOSWebAuthenticationSession.static() {
    return _staticValue;
  }

  @override
  final String id = IdGenerator.generate();

  @override
  late final WebUri url;

  @override
  late final String? callbackURLScheme;

  @override
  late final WebAuthenticationSessionSettings? initialSettings;

  @override
  late final WebAuthenticationSessionCompletionHandler onComplete;

  static const MethodChannel _staticChannel = const MethodChannel(
      'com.pichillilorenzo/flutter_webauthenticationsession');

  @override
  Future<MacOSWebAuthenticationSession> create(
      {required WebUri url,
      String? callbackURLScheme,
      WebAuthenticationSessionCompletionHandler onComplete,
      WebAuthenticationSessionSettings? initialSettings}) async {
    var session = MacOSWebAuthenticationSession._create(
        url: url,
        callbackURLScheme: callbackURLScheme,
        onComplete: onComplete,
        initialSettings: initialSettings);
    initialSettings =
        session.initialSettings ?? WebAuthenticationSessionSettings();
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("id", () => session.id);
    args.putIfAbsent("url", () => session.url.toString());
    args.putIfAbsent("callbackURLScheme", () => session.callbackURLScheme);
    args.putIfAbsent("initialSettings", () => initialSettings?.toMap());
    await _staticChannel.invokeMethod('create', args);
    return session;
  }

  MacOSWebAuthenticationSession._create(
      {required this.url,
      this.callbackURLScheme,
      this.onComplete,
      WebAuthenticationSessionSettings? initialSettings})
      : super.implementation(MacOSWebAuthenticationSessionCreationParams()) {
    assert(url.toString().isNotEmpty);
    assert(['http', 'https'].contains(url.scheme),
        'The specified URL has an unsupported scheme. Only HTTP and HTTPS URLs are supported on iOS.');

    this.initialSettings =
        initialSettings ?? WebAuthenticationSessionSettings();
    channel = MethodChannel(
        'com.pichillilorenzo/flutter_webauthenticationsession_$id');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  _debugLog(String method, dynamic args) {
    debugLog(
        className: this.runtimeType.toString(),
        debugLoggingSettings:
            PlatformWebAuthenticationSession.debugLoggingSettings,
        id: id,
        method: method,
        args: args);
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    _debugLog(call.method, call.arguments);

    switch (call.method) {
      case "onComplete":
        String? url = call.arguments["url"];
        WebUri? uri = url != null ? WebUri(url) : null;
        var error = WebAuthenticationSessionError.fromNativeValue(
            call.arguments["errorCode"]);
        if (onComplete != null) {
          onComplete!(uri, error);
        }
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
  }

  @override
  Future<bool> canStart() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('canStart', args) ?? false;
  }

  @override
  Future<bool> start() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await channel?.invokeMethod<bool>('start', args) ?? false;
  }

  @override
  Future<void> cancel() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod("cancel", args);
  }

  @override
  Future<void> dispose() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod("dispose", args);
    disposeChannel();
  }

  @override
  Future<bool> isAvailable() async {
    Map<String, dynamic> args = <String, dynamic>{};
    return await _staticChannel.invokeMethod<bool>("isAvailable", args) ??
        false;
  }
}
