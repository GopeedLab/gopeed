import 'package:flutter/foundation.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

import 'web_message_channel.dart';

/// Object specifying creation parameters for creating a [MacOSWebMessagePort].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebMessagePortCreationParams] for
/// more information.
@immutable
class MacOSWebMessagePortCreationParams
    extends PlatformWebMessagePortCreationParams {
  /// Creates a new [MacOSWebMessagePortCreationParams] instance.
  const MacOSWebMessagePortCreationParams({required super.index});

  /// Creates a [MacOSWebMessagePortCreationParams] instance based on [PlatformWebMessagePortCreationParams].
  factory MacOSWebMessagePortCreationParams.fromPlatformWebMessagePortCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformWebMessagePortCreationParams params) {
    return MacOSWebMessagePortCreationParams(index: params.index);
  }

  @override
  String toString() {
    return 'MacOSWebMessagePortCreationParams{index: $index}';
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebMessagePort}
class MacOSWebMessagePort extends PlatformWebMessagePort {
  WebMessageCallback? _onMessage;
  late MacOSWebMessageChannel _webMessageChannel;

  /// Constructs a [MacOSWebMessagePort].
  MacOSWebMessagePort(PlatformWebMessagePortCreationParams params)
      : super.implementation(
          params is MacOSWebMessagePortCreationParams
              ? params
              : MacOSWebMessagePortCreationParams
                  .fromPlatformWebMessagePortCreationParams(params),
        );

  @override
  Future<void> setWebMessageCallback(WebMessageCallback? onMessage) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('index', () => params.index);
    await _webMessageChannel.internalChannel
        ?.invokeMethod('setWebMessageCallback', args);
    this._onMessage = onMessage;
  }

  @override
  Future<void> postMessage(WebMessage message) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('index', () => params.index);
    args.putIfAbsent('message', () => message.toMap());
    await _webMessageChannel.internalChannel?.invokeMethod('postMessage', args);
  }

  @override
  Future<void> close() async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('index', () => params.index);
    await _webMessageChannel.internalChannel?.invokeMethod('close', args);
  }

  @override
  Map<String, dynamic> toMap() {
    return {
      "index": params.index,
      "webMessageChannelId": this._webMessageChannel.params.id
    };
  }

  @override
  Map<String, dynamic> toJson() {
    return toMap();
  }

  @override
  String toString() {
    return 'MacOSWebMessagePort{index: ${params.index}}';
  }
}

extension InternalWebMessagePort on MacOSWebMessagePort {
  WebMessageCallback? get onMessage => _onMessage;
  void set onMessage(WebMessageCallback? value) => _onMessage = value;

  MacOSWebMessageChannel get webMessageChannel => _webMessageChannel;
  void set webMessageChannel(MacOSWebMessageChannel value) =>
      _webMessageChannel = value;
}
