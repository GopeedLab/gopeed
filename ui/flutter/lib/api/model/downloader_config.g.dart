// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'downloader_config.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DownloaderConfig _$DownloaderConfigFromJson(Map<String, dynamic> json) =>
    DownloaderConfig()
      ..downloadDir = json['downloadDir'] as String
      ..protocolConfig = ProtocolConfig.fromJson(
          json['protocolConfig'] as Map<String, dynamic>)
      ..extra = ExtraConfig.fromJson(json['extra'] as Map<String, dynamic>);

Map<String, dynamic> _$DownloaderConfigToJson(DownloaderConfig instance) =>
    <String, dynamic>{
      'downloadDir': instance.downloadDir,
      'protocolConfig': instance.protocolConfig.toJson(),
      'extra': instance.extra.toJson(),
    };

ProtocolConfig _$ProtocolConfigFromJson(Map<String, dynamic> json) =>
    ProtocolConfig()
      ..http = HttpConfig.fromJson(json['http'] as Map<String, dynamic>)
      ..bt = BtConfig.fromJson(json['bt'] as Map<String, dynamic>);

Map<String, dynamic> _$ProtocolConfigToJson(ProtocolConfig instance) =>
    <String, dynamic>{
      'http': instance.http.toJson(),
      'bt': instance.bt.toJson(),
    };

HttpConfig _$HttpConfigFromJson(Map<String, dynamic> json) =>
    HttpConfig()..connections = json['connections'] as int;

Map<String, dynamic> _$HttpConfigToJson(HttpConfig instance) =>
    <String, dynamic>{
      'connections': instance.connections,
    };

BtConfig _$BtConfigFromJson(Map<String, dynamic> json) => BtConfig()
  ..trackers =
      (json['trackers'] as List<dynamic>).map((e) => e as String).toList();

Map<String, dynamic> _$BtConfigToJson(BtConfig instance) => <String, dynamic>{
      'trackers': instance.trackers,
    };

ExtraConfig _$ExtraConfigFromJson(Map<String, dynamic> json) => ExtraConfig()
  ..themeMode = json['themeMode'] as String
  ..locale = json['locale'] as String
  ..bt = ExtraConfigBt.fromJson(json['bt'] as Map<String, dynamic>);

Map<String, dynamic> _$ExtraConfigToJson(ExtraConfig instance) =>
    <String, dynamic>{
      'themeMode': instance.themeMode,
      'locale': instance.locale,
      'bt': instance.bt.toJson(),
    };

ExtraConfigBt _$ExtraConfigBtFromJson(Map<String, dynamic> json) =>
    ExtraConfigBt()
      ..trackerSubscribeUrls = (json['trackerSubscribeUrls'] as List<dynamic>)
          .map((e) => e as String)
          .toList()
      ..subscribeTrackers = (json['subscribeTrackers'] as List<dynamic>)
          .map((e) => e as String)
          .toList()
      ..lastTrackerUpdateTime = json['lastTrackerUpdateTime'] == null
          ? null
          : DateTime.parse(json['lastTrackerUpdateTime'] as String)
      ..customTrackers = (json['customTrackers'] as List<dynamic>)
          .map((e) => e as String)
          .toList();

Map<String, dynamic> _$ExtraConfigBtToJson(ExtraConfigBt instance) {
  final val = <String, dynamic>{
    'trackerSubscribeUrls': instance.trackerSubscribeUrls,
    'subscribeTrackers': instance.subscribeTrackers,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('lastTrackerUpdateTime',
      instance.lastTrackerUpdateTime?.toIso8601String());
  val['customTrackers'] = instance.customTrackers;
  return val;
}
