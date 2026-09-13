import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';
import 'web_message_port.dart';

/// Object specifying creation parameters for creating a [IOSWebMessageChannel].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebMessageChannelCreationParams] for
/// more information.
@immutable
class IOSWebMessageChannelCreationParams
    extends PlatformWebMessageChannelCreationParams {
  /// Creates a new [IOSWebMessageChannelCreationParams] instance.
  const IOSWebMessageChannelCreationParams(
      {required super.id, required super.port1, required super.port2});

  /// Creates a [IOSWebMessageChannelCreationParams] instance based on [PlatformWebMessageChannelCreationParams].
  factory IOSWebMessageChannelCreationParams.fromPlatformWebMessageChannelCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformWebMessageChannelCreationParams params) {
    return IOSWebMessageChannelCreationParams(
        id: params.id, port1: params.port1, port2: params.port2);
  }

  @override
  String toString() {
    return 'IOSWebMessageChannelCreationParams{id: $id, port1: $port1, port2: $port2}';
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebMessageChannel}
class IOSWebMessageChannel extends PlatformWebMessageChannel
    with ChannelController {
  /// Constructs a [IOSWebMessageChannel].
  IOSWebMessageChannel(PlatformWebMessageChannelCreationParams params)
      : super.implementation(
          params is IOSWebMessageChannelCreationParams
              ? params
              : IOSWebMessageChannelCreationParams
                  .fromPlatformWebMessageChannelCreationParams(params),
        ) {
    channel = MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_web_message_channel_${params.id}');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  static final IOSWebMessageChannel _staticValue = IOSWebMessageChannel(
      IOSWebMessageChannelCreationParams(
          id: '',
          port1: IOSWebMessagePort(IOSWebMessagePortCreationParams(index: 0)),
          port2: IOSWebMessagePort(IOSWebMessagePortCreationParams(index: 1))));

  /// Provide static access.
  factory IOSWebMessageChannel.static() {
    return _staticValue;
  }

  IOSWebMessagePort get _iosPort1 => port1 as IOSWebMessagePort;

  IOSWebMessagePort get _iosPort2 => port2 as IOSWebMessagePort;

  static IOSWebMessageChannel? _fromMap(Map<String, dynamic>? map) {
    if (map == null) {
      return null;
    }
    var webMessageChannel = IOSWebMessageChannel(
        IOSWebMessageChannelCreationParams(
            id: map["id"],
            port1: IOSWebMessagePort(IOSWebMessagePortCreationParams(index: 0)),
            port2:
                IOSWebMessagePort(IOSWebMessagePortCreationParams(index: 1))));
    webMessageChannel._iosPort1.webMessageChannel = webMessageChannel;
    webMessageChannel._iosPort2.webMessageChannel = webMessageChannel;
    return webMessageChannel;
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    switch (call.method) {
      case "onMessage":
        int index = call.arguments["index"];
        var port = index == 0 ? _iosPort1 : _iosPort2;
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
  IOSWebMessageChannel? fromMap(Map<String, dynamic>? map) {
    return _fromMap(map);
  }

  @override
  void dispose() {
    disposeChannel();
  }

  @override
  String toString() {
    return 'IOSWebMessageChannel{id: $id, port1: $port1, port2: $port2}';
  }
}

extension InternalWebMessageChannel on IOSWebMessageChannel {
  MethodChannel? get internalChannel => channel;
}
