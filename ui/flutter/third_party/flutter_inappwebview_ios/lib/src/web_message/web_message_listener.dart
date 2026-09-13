import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [IOSWebMessageListener].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebMessageListenerCreationParams] for
/// more information.
@immutable
class IOSWebMessageListenerCreationParams
    extends PlatformWebMessageListenerCreationParams {
  /// Creates a new [IOSWebMessageListenerCreationParams] instance.
  const IOSWebMessageListenerCreationParams(
      {required this.allowedOriginRules,
      required super.jsObjectName,
      super.onPostMessage});

  /// Creates a [IOSWebMessageListenerCreationParams] instance based on [PlatformWebMessageListenerCreationParams].
  factory IOSWebMessageListenerCreationParams.fromPlatformWebMessageListenerCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformWebMessageListenerCreationParams params) {
    return IOSWebMessageListenerCreationParams(
        allowedOriginRules: params.allowedOriginRules ?? Set.from(["*"]),
        jsObjectName: params.jsObjectName,
        onPostMessage: params.onPostMessage);
  }

  @override
  final Set<String> allowedOriginRules;

  @override
  String toString() {
    return 'IOSWebMessageListenerCreationParams{jsObjectName: $jsObjectName, allowedOriginRules: $allowedOriginRules, onPostMessage: $onPostMessage}';
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebMessageListener}
class IOSWebMessageListener extends PlatformWebMessageListener
    with ChannelController {
  /// Constructs a [IOSWebMessageListener].
  IOSWebMessageListener(PlatformWebMessageListenerCreationParams params)
      : super.implementation(
          params is IOSWebMessageListenerCreationParams
              ? params
              : IOSWebMessageListenerCreationParams
                  .fromPlatformWebMessageListenerCreationParams(params),
        ) {
    assert(!this._iosParams.allowedOriginRules.contains(""),
        "allowedOriginRules cannot contain empty strings");
    channel = MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_web_message_listener_${_id}_${params.jsObjectName}');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  ///Message Listener ID used internally.
  final String _id = IdGenerator.generate();

  IOSJavaScriptReplyProxy? _replyProxy;

  IOSWebMessageListenerCreationParams get _iosParams =>
      params as IOSWebMessageListenerCreationParams;

  Future<dynamic> _handleMethod(MethodCall call) async {
    switch (call.method) {
      case "onPostMessage":
        if (_replyProxy == null) {
          _replyProxy = IOSJavaScriptReplyProxy(
              PlatformJavaScriptReplyProxyCreationParams(
                  webMessageListener: this));
        }
        if (onPostMessage != null) {
          WebMessage? message = call.arguments["message"] != null
              ? WebMessage.fromMap(
                  call.arguments["message"].cast<String, dynamic>())
              : null;
          WebUri? sourceOrigin = call.arguments["sourceOrigin"] != null
              ? WebUri(call.arguments["sourceOrigin"])
              : null;
          bool isMainFrame = call.arguments["isMainFrame"];
          onPostMessage!(message, sourceOrigin, isMainFrame, _replyProxy!);
        }
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
    return null;
  }

  @override
  void dispose() {
    disposeChannel();
  }

  @override
  Map<String, dynamic> toMap() {
    return {
      "id": _id,
      "jsObjectName": params.jsObjectName,
      "allowedOriginRules": _iosParams.allowedOriginRules.toList(),
    };
  }

  @override
  Map<String, dynamic> toJson() {
    return this.toMap();
  }

  @override
  String toString() {
    return 'IOSWebMessageListener{id: ${_id}, jsObjectName: ${params.jsObjectName}, allowedOriginRules: ${params.allowedOriginRules}, replyProxy: $_replyProxy}';
  }
}

/// Object specifying creation parameters for creating a [IOSJavaScriptReplyProxy].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformJavaScriptReplyProxyCreationParams] for
/// more information.
@immutable
class IOSJavaScriptReplyProxyCreationParams
    extends PlatformJavaScriptReplyProxyCreationParams {
  /// Creates a new [IOSJavaScriptReplyProxyCreationParams] instance.
  const IOSJavaScriptReplyProxyCreationParams(
      {required super.webMessageListener});

  /// Creates a [IOSJavaScriptReplyProxyCreationParams] instance based on [PlatformJavaScriptReplyProxyCreationParams].
  factory IOSJavaScriptReplyProxyCreationParams.fromPlatformJavaScriptReplyProxyCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformJavaScriptReplyProxyCreationParams params) {
    return IOSJavaScriptReplyProxyCreationParams(
        webMessageListener: params.webMessageListener);
  }
}

///{@macro flutter_inappwebview_platform_interface.JavaScriptReplyProxy}
class IOSJavaScriptReplyProxy extends PlatformJavaScriptReplyProxy {
  /// Constructs a [IOSWebMessageListener].
  IOSJavaScriptReplyProxy(PlatformJavaScriptReplyProxyCreationParams params)
      : super.implementation(
          params is IOSJavaScriptReplyProxyCreationParams
              ? params
              : IOSJavaScriptReplyProxyCreationParams
                  .fromPlatformJavaScriptReplyProxyCreationParams(params),
        );

  IOSWebMessageListener get _iosWebMessageListener =>
      params.webMessageListener as IOSWebMessageListener;

  @override
  Future<void> postMessage(WebMessage message) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('message', () => message.toMap());
    await _iosWebMessageListener.channel?.invokeMethod('postMessage', args);
  }

  @override
  String toString() {
    return 'IOSJavaScriptReplyProxy{}';
  }
}
