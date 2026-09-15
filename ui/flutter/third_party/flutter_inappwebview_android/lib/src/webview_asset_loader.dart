import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [AndroidPathHandler].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformPathHandlerCreationParams] for
/// more information.
@immutable
class AndroidPathHandlerCreationParams
    extends PlatformPathHandlerCreationParams {
  /// Creates a new [AndroidPathHandlerCreationParams] instance.
  AndroidPathHandlerCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformPathHandlerCreationParams params,
  ) : super(path: params.path);

  /// Creates a [AndroidPathHandlerCreationParams] instance based on [PlatformPathHandlerCreationParams].
  factory AndroidPathHandlerCreationParams.fromPlatformPathHandlerCreationParams(
      PlatformPathHandlerCreationParams params) {
    return AndroidPathHandlerCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformPathHandler}
abstract mixin class AndroidPathHandler
    implements ChannelController, PlatformPathHandler {
  final String _id = IdGenerator.generate();

  @override
  late final PlatformPathHandlerEvents? eventHandler;

  @override
  late final String path;

  void _init(PlatformPathHandlerCreationParams params) {
    this.path = params.path;
    channel = MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_custompathhandler_${_id}');
    handler = _handleMethod;
    initMethodCallHandler();
  }

  Future<dynamic> _handleMethod(MethodCall call) async {
    switch (call.method) {
      case "handle":
        String path = call.arguments["path"];
        return (await eventHandler?.handle(path))?.toMap();
      default:
        throw UnimplementedError("Unimplemented ${call.method} method");
    }
  }

  @override
  Map<String, dynamic> toMap() {
    return {"path": path, "type": type, "id": _id};
  }

  @override
  Map<String, dynamic> toJson() {
    return toMap();
  }

  @override
  String toString() {
    return 'AndroidPathHandler{path: $path, type: $type}';
  }

  @override
  void dispose() {
    disposeChannel();
    eventHandler = null;
  }
}

/// Object specifying creation parameters for creating a [AndroidAssetsPathHandler].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformAssetsPathHandlerCreationParams] for
/// more information.
@immutable
class AndroidAssetsPathHandlerCreationParams
    extends PlatformAssetsPathHandlerCreationParams {
  /// Creates a new [AndroidAssetsPathHandlerCreationParams] instance.
  AndroidAssetsPathHandlerCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformAssetsPathHandlerCreationParams params,
  ) : super(params);

  /// Creates a [AndroidAssetsPathHandlerCreationParams] instance based on [PlatformAssetsPathHandlerCreationParams].
  factory AndroidAssetsPathHandlerCreationParams.fromPlatformAssetsPathHandlerCreationParams(
      PlatformAssetsPathHandlerCreationParams params) {
    return AndroidAssetsPathHandlerCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformAssetsPathHandler}
class AndroidAssetsPathHandler extends PlatformAssetsPathHandler
    with AndroidPathHandler, ChannelController {
  /// Constructs a [AndroidAssetsPathHandler].
  AndroidAssetsPathHandler(PlatformAssetsPathHandlerCreationParams params)
      : super.implementation(
          params is AndroidAssetsPathHandlerCreationParams
              ? params
              : AndroidAssetsPathHandlerCreationParams
                  .fromPlatformAssetsPathHandlerCreationParams(params),
        ) {
    _init(params);
  }
}

/// Object specifying creation parameters for creating a [AndroidResourcesPathHandler].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformResourcesPathHandlerCreationParams] for
/// more information.
@immutable
class AndroidResourcesPathHandlerCreationParams
    extends PlatformResourcesPathHandlerCreationParams {
  /// Creates a new [AndroidResourcesPathHandlerCreationParams] instance.
  AndroidResourcesPathHandlerCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformResourcesPathHandlerCreationParams params,
  ) : super(params);

  /// Creates a [AndroidResourcesPathHandlerCreationParams] instance based on [PlatformResourcesPathHandlerCreationParams].
  factory AndroidResourcesPathHandlerCreationParams.fromPlatformResourcesPathHandlerCreationParams(
      PlatformResourcesPathHandlerCreationParams params) {
    return AndroidResourcesPathHandlerCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformResourcesPathHandler}
class AndroidResourcesPathHandler extends PlatformResourcesPathHandler
    with AndroidPathHandler, ChannelController {
  /// Constructs a [AndroidResourcesPathHandler].
  AndroidResourcesPathHandler(PlatformResourcesPathHandlerCreationParams params)
      : super.implementation(
          params is AndroidResourcesPathHandlerCreationParams
              ? params
              : AndroidResourcesPathHandlerCreationParams
                  .fromPlatformResourcesPathHandlerCreationParams(params),
        ) {
    _init(params);
  }
}

/// Object specifying creation parameters for creating a [AndroidInternalStoragePathHandler].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformInternalStoragePathHandlerCreationParams] for
/// more information.
@immutable
class AndroidInternalStoragePathHandlerCreationParams
    extends PlatformInternalStoragePathHandlerCreationParams {
  /// Creates a new [AndroidInternalStoragePathHandlerCreationParams] instance.
  AndroidInternalStoragePathHandlerCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformInternalStoragePathHandlerCreationParams params,
  ) : super(params, directory: params.directory);

  /// Creates a [AndroidInternalStoragePathHandlerCreationParams] instance based on [PlatformInternalStoragePathHandlerCreationParams].
  factory AndroidInternalStoragePathHandlerCreationParams.fromPlatformInternalStoragePathHandlerCreationParams(
      PlatformInternalStoragePathHandlerCreationParams params) {
    return AndroidInternalStoragePathHandlerCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformInternalStoragePathHandler}
class AndroidInternalStoragePathHandler
    extends PlatformInternalStoragePathHandler
    with AndroidPathHandler, ChannelController {
  /// Constructs a [AndroidInternalStoragePathHandler].
  AndroidInternalStoragePathHandler(
      PlatformInternalStoragePathHandlerCreationParams params)
      : super.implementation(
          params is AndroidInternalStoragePathHandlerCreationParams
              ? params
              : AndroidInternalStoragePathHandlerCreationParams
                  .fromPlatformInternalStoragePathHandlerCreationParams(params),
        ) {
    _init(params);
  }

  AndroidInternalStoragePathHandlerCreationParams get _internalParams =>
      params as AndroidInternalStoragePathHandlerCreationParams;

  @override
  String get directory => _internalParams.directory;

  @override
  Map<String, dynamic> toMap() {
    return {...toMap(), 'directory': directory};
  }
}

/// Object specifying creation parameters for creating a [AndroidCustomPathHandler].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformCustomPathHandlerCreationParams] for
/// more information.
@immutable
class AndroidCustomPathHandlerCreationParams
    extends PlatformCustomPathHandlerCreationParams {
  /// Creates a new [AndroidCustomPathHandlerCreationParams] instance.
  AndroidCustomPathHandlerCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformCustomPathHandlerCreationParams params,
  ) : super(params);

  /// Creates a [AndroidCustomPathHandlerCreationParams] instance based on [PlatformCustomPathHandlerCreationParams].
  factory AndroidCustomPathHandlerCreationParams.fromPlatformCustomPathHandlerCreationParams(
      PlatformCustomPathHandlerCreationParams params) {
    return AndroidCustomPathHandlerCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformCustomPathHandler}
class AndroidCustomPathHandler extends PlatformCustomPathHandler
    with AndroidPathHandler, ChannelController {
  /// Constructs a [AndroidCustomPathHandler].
  AndroidCustomPathHandler(PlatformCustomPathHandlerCreationParams params)
      : super.implementation(
          params is AndroidCustomPathHandlerCreationParams
              ? params
              : AndroidCustomPathHandlerCreationParams
                  .fromPlatformCustomPathHandlerCreationParams(params),
        ) {
    _init(params);
  }
}
