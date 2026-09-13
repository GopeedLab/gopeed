import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [MacOSPrintJobController].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformPrintJobControllerCreationParams] for
/// more information.
@immutable
class MacOSPrintJobControllerCreationParams
    extends PlatformPrintJobControllerCreationParams {
  /// Creates a new [MacOSPrintJobControllerCreationParams] instance.
  const MacOSPrintJobControllerCreationParams(
      {required super.id, super.onComplete});

  /// Creates a [MacOSPrintJobControllerCreationParams] instance based on [PlatformPrintJobControllerCreationParams].
  factory MacOSPrintJobControllerCreationParams.fromPlatformPrintJobControllerCreationParams(
      // Recommended placeholder to prevent being broken by platform interface.
      // ignore: avoid_unused_constructor_parameters
      PlatformPrintJobControllerCreationParams params) {
    return MacOSPrintJobControllerCreationParams(
        id: params.id, onComplete: params.onComplete);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformPrintJobController}
class MacOSPrintJobController extends PlatformPrintJobController
    with ChannelController {
  /// Constructs a [MacOSPrintJobController].
  MacOSPrintJobController(PlatformPrintJobControllerCreationParams params)
      : super.implementation(
          params is MacOSPrintJobControllerCreationParams
              ? params
              : MacOSPrintJobControllerCreationParams
                  .fromPlatformPrintJobControllerCreationParams(params),
        ) {
    onComplete = params.onComplete;
    channel = MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_printjobcontroller_${params.id}');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    switch (call.method) {
      case "onComplete":
        bool completed = call.arguments["completed"];
        String? error = call.arguments["error"];
        if (onComplete != null) {
          onComplete!(completed, error);
        }
        break;
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
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
