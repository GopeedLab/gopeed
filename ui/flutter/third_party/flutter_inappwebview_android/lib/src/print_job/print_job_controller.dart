import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [AndroidPrintJobController].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformPrintJobControllerCreationParams] for
/// more information.
@immutable
class AndroidPrintJobControllerCreationParams
    extends PlatformPrintJobControllerCreationParams {
  /// Creates a new [AndroidPrintJobControllerCreationParams] instance.
  const AndroidPrintJobControllerCreationParams(
      {required super.id, super.onComplete});

  /// Creates a [AndroidPrintJobControllerCreationParams] instance based on [PlatformPrintJobControllerCreationParams].
  factory AndroidPrintJobControllerCreationParams.fromPlatformPrintJobControllerCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformPrintJobControllerCreationParams params) {
    return AndroidPrintJobControllerCreationParams(
        id: params.id, onComplete: params.onComplete);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformPrintJobController}
class AndroidPrintJobController extends PlatformPrintJobController
    with ChannelController {
  /// Constructs a [AndroidPrintJobController].
  AndroidPrintJobController(PlatformPrintJobControllerCreationParams params)
      : super.implementation(
          params is AndroidPrintJobControllerCreationParams
              ? params
              : AndroidPrintJobControllerCreationParams
                  .fromPlatformPrintJobControllerCreationParams(params),
        ) {
    channel = MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_printjobcontroller_${params.id}');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    switch (call.method) {
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
  }

  @override
  Future<void> cancel() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('cancel', args);
  }

  @override
  Future<void> restart() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('restart', args);
  }

  @override
  Future<PrintJobInfo?> getInfo() async {
    Map<String, dynamic> args = <String, dynamic>{};
    Map<String, dynamic>? infoMap =
        (await channel?.invokeMethod('getInfo', args))?.cast<String, dynamic>();
    return PrintJobInfo.fromMap(infoMap);
  }

  @override
  Future<void> dispose() async {
    Map<String, dynamic> args = <String, dynamic>{};
    await channel?.invokeMethod('dispose', args);
    disposeChannel();
  }
}
