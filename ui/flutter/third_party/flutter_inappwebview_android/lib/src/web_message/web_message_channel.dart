import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';
import 'web_message_port.dart';

/// Object specifying creation parameters for creating a [AndroidWebMessageChannel].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebMessageChannelCreationParams] for
/// more information.
@immutable
class AndroidWebMessageChannelCreationParams
    extends PlatformWebMessageChannelCreationParams {
  /// Creates a new [AndroidWebMessageChannelCreationParams] instance.
  const AndroidWebMessageChannelCreationParams(
      {required super.id, required super.port1, required super.port2});

  /// Creates a [AndroidWebMessageChannelCreationParams] instance based on [PlatformWebMessageChannelCreationParams].
  factory AndroidWebMessageChannelCreationParams.fromPlatformWebMessageChannelCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformWebMessageChannelCreationParams params) {
    return AndroidWebMessageChannelCreationParams(
        id: params.id, port1: params.port1, port2: params.port2);
  }

  @override
  String toString() {
    return 'AndroidWebMessageChannelCreationParams{id: $id, port1: $port1, port2: $port2}';
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebMessageChannel}
class AndroidWebMessageChannel extends PlatformWebMessageChannel
    with ChannelController {
  /// Constructs a [AndroidWebMessageChannel].
  AndroidWebMessageChannel(PlatformWebMessageChannelCreationParams params)
      : super.implementation(
          params is AndroidWebMessageChannelCreationParams
              ? params
              : AndroidWebMessageChannelCreationParams
                  .fromPlatformWebMessageChannelCreationParams(params),
        ) {
    channel = MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_web_message_channel_${params.id}');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  static final AndroidWebMessageChannel _staticValue = AndroidWebMessageChannel(
      AndroidWebMessageChannelCreationParams(
          id: '',
          port1: AndroidWebMessagePort(
              AndroidWebMessagePortCreationParams(index: 0)),
          port2: AndroidWebMessagePort(
              AndroidWebMessagePortCreationParams(index: 1))));

  /// Provide static access.
  factory AndroidWebMessageChannel.static() {
    return _staticValue;
  }

  AndroidWebMessagePort get _androidPort1 => port1 as AndroidWebMessagePort;

  AndroidWebMessagePort get _androidPort2 => port2 as AndroidWebMessagePort;

  static AndroidWebMessageChannel? _fromMap(Map<String, dynamic>? map) {
    if (map == null) {
      return null;
    }
    var webMessageChannel = AndroidWebMessageChannel(
        AndroidWebMessageChannelCreationParams(
            id: map["id"],
            port1: AndroidWebMessagePort(
                AndroidWebMessagePortCreationParams(index: 0)),
            port2: AndroidWebMessagePort(
                AndroidWebMessagePortCreationParams(index: 1))));
    webMessageChannel._androidPort1.webMessageChannel = webMessageChannel;
    webMessageChannel._androidPort2.webMessageChannel = webMessageChannel;
    return webMessageChannel;
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    switch (call.method) {
      case "onMessage":
        int index = call.arguments["index"];
        var port = index == 0 ? _androidPort1 : _androidPort2;
        if (port.onMessage != null) {
          WebMessage? message = call.arguments["message"] != null
              ? WebMessage.fromMap(
                  call.arguments["message"].cast<String, dynamic>())
              : null;
          port.onMessage!(message);
        }
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
    return null;
  }

  @override
  AndroidWebMessageChannel? fromMap(Map<String, dynamic>? map) {
    return _fromMap(map);
  }

  @override
  void dispose() {
    disposeChannel();
  }

  @override
  String toString() {
    return 'AndroidWebMessageChannel{id: $id, port1: $port1, port2: $port2}';
  }
}

extension InternalWebMessageChannel on AndroidWebMessageChannel {
  MethodChannel? get internalChannel => channel;
}
