import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_inappwebview_platform_interface/flutter_inappwebview_platform_interface.dart';

/// Object specifying creation parameters for creating a [IOSWebStorageManager].
///
/// When adding additional fields make sure they can be null or have a default
/// value to avoid breaking changes. See [PlatformWebStorageManagerCreationParams] for
/// more information.
@immutable
class IOSWebStorageManagerCreationParams
    extends PlatformWebStorageManagerCreationParams {
  /// Creates a new [IOSWebStorageManagerCreationParams] instance.
  const IOSWebStorageManagerCreationParams(
    // This parameter prevents breaking changes later.
    // ignore: avoid_unused_constructor_parameters
    PlatformWebStorageManagerCreationParams params,
  ) : super();

  /// Creates a [IOSWebStorageManagerCreationParams] instance based on [PlatformWebStorageManagerCreationParams].
  factory IOSWebStorageManagerCreationParams.fromPlatformWebStorageManagerCreationParams(
      PlatformWebStorageManagerCreationParams params) {
    return IOSWebStorageManagerCreationParams(params);
  }
}

///{@macro flutter_inappwebview_platform_interface.PlatformWebStorageManager}
class IOSWebStorageManager extends PlatformWebStorageManager
    with ChannelController {
  /// Creates a new [IOSWebStorageManager].
  IOSWebStorageManager(PlatformWebStorageManagerCreationParams params)
      : super.implementation(
          params is IOSWebStorageManagerCreationParams
              ? params
              : IOSWebStorageManagerCreationParams
                  .fromPlatformWebStorageManagerCreationParams(params),
        ) {
    channel = const MethodChannel(
        'com.pichillilorenzo/flutter_inappwebview_webstoragemanager');
    handler = handleMethod;
    initMethodCallHandler();
  }

  static IOSWebStorageManager? _instance;

  ///Gets the WebStorage manager shared instance.
  static IOSWebStorageManager instance() {
    return (_instance != null) ? _instance! : _init();
  }

  static IOSWebStorageManager _init() {
    _instance = IOSWebStorageManager(IOSWebStorageManagerCreationParams(
        const PlatformWebStorageManagerCreationParams()));
    return _instance!;
  }

  Future<dynamic> _handleMethod(MethodCall call) async {}

  @override
  Future<List<WebsiteDataRecord>> fetchDataRecords(
      {required Set<WebsiteDataType> dataTypes}) async {
    List<WebsiteDataRecord> recordList = [];
    List<String> dataTypesList = [];
    for (var dataType in dataTypes) {
      dataTypesList.add(dataType.toNativeValue());
    }
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("dataTypes", () => dataTypesList);
    List<Map<dynamic, dynamic>> records =
        (await channel?.invokeMethod<List>('fetchDataRecords', args))
                ?.cast<Map<dynamic, dynamic>>() ??
            [];
    for (var record in records) {
      List<String> dataTypesString = record["dataTypes"].cast<String>();
      Set<WebsiteDataType> dataTypes = Set();
      for (var dataTypeValue in dataTypesString) {
        var dataType = WebsiteDataType.fromNativeValue(dataTypeValue);
        if (dataType != null) {
          dataTypes.add(dataType);
        }
      }
      recordList.add(WebsiteDataRecord(
          displayName: record["displayName"], dataTypes: dataTypes));
    }
    return recordList;
  }

  @override
  Future<void> removeDataFor(
      {required Set<WebsiteDataType> dataTypes,
      required List<WebsiteDataRecord> dataRecords}) async {
    List<String> dataTypesList = [];
    for (var dataType in dataTypes) {
      dataTypesList.add(dataType.toNativeValue());
    }

    List<Map<String, dynamic>> recordList = [];
    for (var record in dataRecords) {
      recordList.add(record.toMap());
    }

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("dataTypes", () => dataTypesList);
    args.putIfAbsent("recordList", () => recordList);
    await channel?.invokeMethod('removeDataFor', args);
  }

  @override
  Future<void> removeDataModifiedSince(
      {required Set<WebsiteDataType> dataTypes, required DateTime date}) async {
    List<String> dataTypesList = [];
    for (var dataType in dataTypes) {
      dataTypesList.add(dataType.toNativeValue());
    }

    var timestamp = date.millisecondsSinceEpoch;

    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent("dataTypes", () => dataTypesList);
    args.putIfAbsent("timestamp", () => timestamp);
    await channel?.invokeMethod('removeDataModifiedSince', args);
  }

  @override
  void dispose() {
    // empty
  }
}

extension InternalWebStorageManager on IOSWebStorageManager {
  get handleMethod => _handleMethod;
}
