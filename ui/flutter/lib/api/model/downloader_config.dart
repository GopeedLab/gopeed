import 'package:json_annotation/json_annotation.dart';

part 'downloader_config.g.dart';

@JsonSerializable(explicitToJson: true)
class DownloaderConfig {
  String downloadDir = '';
  int maxRunning = 0;
  ProtocolConfig protocolConfig = ProtocolConfig();
  ExtraConfig extra = ExtraConfig();

  DownloaderConfig();

  factory DownloaderConfig.fromJson(Map<String, dynamic> json) =>
      _$DownloaderConfigFromJson(json);
  Map<String, dynamic> toJson() => _$DownloaderConfigToJson(this);
}

@JsonSerializable(explicitToJson: true)
class ProtocolConfig {
  HttpConfig http = HttpConfig();
  BtConfig bt = BtConfig();

  ProtocolConfig();

  factory ProtocolConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ProtocolConfig() : _$ProtocolConfigFromJson(json);
  Map<String, dynamic> toJson() => _$ProtocolConfigToJson(this);
}

@JsonSerializable()
class HttpConfig {
  String userAgent =
      'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36';
  int connections = 0;

  HttpConfig();

  factory HttpConfig.fromJson(Map<String, dynamic> json) =>
      _$HttpConfigFromJson(json);
  Map<String, dynamic> toJson() => _$HttpConfigToJson(this);
}

@JsonSerializable()
class BtConfig {
  int listenPort = 0;
  List<String> trackers = [];

  BtConfig();

  factory BtConfig.fromJson(Map<String, dynamic> json) =>
      _$BtConfigFromJson(json);
  Map<String, dynamic> toJson() => _$BtConfigToJson(this);
}

@JsonSerializable(explicitToJson: true)
class ExtraConfig {
  String themeMode = '';
  String locale = '';
  ExtraConfigBt bt = ExtraConfigBt();

  ExtraConfig();

  factory ExtraConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ExtraConfig() : _$ExtraConfigFromJson(json);
  Map<String, dynamic> toJson() => _$ExtraConfigToJson(this);
}

@JsonSerializable()
class ExtraConfigBt {
  List<String> trackerSubscribeUrls = [];
  List<String> subscribeTrackers = [];
  DateTime? lastTrackerUpdateTime;

  List<String> customTrackers = [];

  ExtraConfigBt();

  factory ExtraConfigBt.fromJson(Map<String, dynamic> json) =>
      _$ExtraConfigBtFromJson(json);
  Map<String, dynamic> toJson() => _$ExtraConfigBtToJson(this);
}
