import 'package:json_annotation/json_annotation.dart';

part 'downloader_config.g.dart';

@JsonSerializable(explicitToJson: true)
class DownloaderConfig {
  String downloadDir = '';
  ProtocolConfig? protocolConfig;
  ExtraConfig? extra;

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

  factory ProtocolConfig.fromJson(Map<String, dynamic> json) =>
      _$ProtocolConfigFromJson(json);
  Map<String, dynamic> toJson() => _$ProtocolConfigToJson(this);
}

@JsonSerializable()
class HttpConfig {
  int connections = 0;

  HttpConfig();

  factory HttpConfig.fromJson(Map<String, dynamic> json) =>
      _$HttpConfigFromJson(json);
  Map<String, dynamic> toJson() => _$HttpConfigToJson(this);
}

@JsonSerializable()
class BtConfig {
  List<String> trackerSubscribeUrls = [];
  List<String> trackers = [];

  BtConfig();

  factory BtConfig.fromJson(Map<String, dynamic> json) =>
      _$BtConfigFromJson(json);
  Map<String, dynamic> toJson() => _$BtConfigToJson(this);
}

@JsonSerializable()
class ExtraConfig {
  String themeMode = '';
  String locale = '';

  ExtraConfig({
    this.themeMode = '',
    this.locale = '',
  });

  factory ExtraConfig.fromJson(Map<String, dynamic> json) =>
      _$ExtraConfigFromJson(json);
  Map<String, dynamic> toJson() => _$ExtraConfigToJson(this);
}
