// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'downloader_config.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DownloaderConfig _$DownloaderConfigFromJson(Map<String, dynamic> json) =>
    DownloaderConfig()
      ..downloadDir = json['downloadDir'] as String
      ..protocolConfig = json['protocolConfig'] == null
          ? null
          : ProtocolConfig.fromJson(
              json['protocolConfig'] as Map<String, dynamic>)
      ..extra = json['extra'] == null
          ? null
          : ExtraConfig.fromJson(json['extra'] as Map<String, dynamic>);

Map<String, dynamic> _$DownloaderConfigToJson(DownloaderConfig instance) {
  final val = <String, dynamic>{
    'downloadDir': instance.downloadDir,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('protocolConfig', instance.protocolConfig?.toJson());
  writeNotNull('extra', instance.extra?.toJson());
  return val;
}

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
  ..trackerSubscribeUrls = (json['trackerSubscribeUrls'] as List<dynamic>)
      .map((e) => e as String)
      .toList()
  ..trackers =
      (json['trackers'] as List<dynamic>).map((e) => e as String).toList();

Map<String, dynamic> _$BtConfigToJson(BtConfig instance) => <String, dynamic>{
      'trackerSubscribeUrls': instance.trackerSubscribeUrls,
      'trackers': instance.trackers,
    };

ExtraConfig _$ExtraConfigFromJson(Map<String, dynamic> json) => ExtraConfig(
      themeMode: json['themeMode'] as String? ?? '',
      locale: json['locale'] as String? ?? '',
    );

Map<String, dynamic> _$ExtraConfigToJson(ExtraConfig instance) =>
    <String, dynamic>{
      'themeMode': instance.themeMode,
      'locale': instance.locale,
    };
