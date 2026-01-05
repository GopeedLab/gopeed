import 'package:json_annotation/json_annotation.dart';

part 'downloader_config.g.dart';

@JsonSerializable(explicitToJson: true)
class DownloaderConfig {
  String downloadDir;
  int maxRunning;
  ProtocolConfig protocolConfig = ProtocolConfig();
  ExtraConfig extra = ExtraConfig();
  ProxyConfig proxy = ProxyConfig();
  WebhookConfig webhook = WebhookConfig();
  ArchiveConfig archive = ArchiveConfig();

  DownloaderConfig({
    this.downloadDir = '',
    this.maxRunning = 0,
  });

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
  String userAgent;
  int connections;
  bool useServerCtime;

  HttpConfig({
    this.userAgent = '',
    this.connections = 0,
    this.useServerCtime = false,
  });

  factory HttpConfig.fromJson(Map<String, dynamic> json) =>
      _$HttpConfigFromJson(json);

  Map<String, dynamic> toJson() => _$HttpConfigToJson(this);
}

@JsonSerializable()
class BtConfig {
  int listenPort;
  List<String> trackers;
  bool seedKeep;
  double seedRatio;
  int seedTime;

  BtConfig({
    this.listenPort = 0,
    this.trackers = const [],
    this.seedKeep = false,
    this.seedRatio = 0,
    this.seedTime = 0,
  });

  factory BtConfig.fromJson(Map<String, dynamic> json) =>
      _$BtConfigFromJson(json);

  Map<String, dynamic> toJson() => _$BtConfigToJson(this);
}

@JsonSerializable(explicitToJson: true)
class ExtraConfig {
  String themeMode;
  String locale;
  bool lastDeleteTaskKeep;
  bool defaultDirectDownload;
  bool defaultBtClient;
  bool notifyWhenNewVersion;
  List<DownloadCategory> downloadCategories;

  ExtraConfigBt bt = ExtraConfigBt();
  ExtraConfigGithubMirror githubMirror = ExtraConfigGithubMirror();

  ExtraConfig({
    this.themeMode = '',
    this.locale = '',
    this.lastDeleteTaskKeep = false,
    this.defaultDirectDownload = false,
    this.defaultBtClient = true,
    this.notifyWhenNewVersion = true,
    this.downloadCategories = const [],
  });

  factory ExtraConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ExtraConfig() : _$ExtraConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ExtraConfigToJson(this);
}

@JsonSerializable()
class DownloadCategory {
  String name;
  String path;
  bool isBuiltIn;
  String? nameKey; // i18n key for built-in categories (e.g., 'categoryMusic')
  bool
      isDeleted; // Mark built-in categories as deleted instead of removing them

  DownloadCategory({
    required this.name,
    required this.path,
    this.isBuiltIn = false,
    this.nameKey,
    this.isDeleted = false,
  });

  factory DownloadCategory.fromJson(Map<String, dynamic> json) =>
      _$DownloadCategoryFromJson(json);

  Map<String, dynamic> toJson() => _$DownloadCategoryToJson(this);
}

@JsonSerializable()
class WebhookConfig {
  bool enable;
  List<String> urls;

  WebhookConfig({
    this.enable = false,
    this.urls = const [],
  });

  factory WebhookConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? WebhookConfig() : _$WebhookConfigFromJson(json);

  Map<String, dynamic> toJson() => _$WebhookConfigToJson(this);
}

@JsonSerializable()
class ProxyConfig {
  bool enable;
  bool system;
  String scheme;
  String host;
  String usr;
  String pwd;

  ProxyConfig({
    this.enable = false,
    this.system = false,
    this.scheme = '',
    this.host = '',
    this.usr = '',
    this.pwd = '',
  });

  factory ProxyConfig.fromJson(Map<String, dynamic> json) =>
      _$ProxyConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ProxyConfigToJson(this);
}

@JsonSerializable()
class ExtraConfigBt {
  List<String> trackerSubscribeUrls = [];
  List<String> subscribeTrackers = [];
  bool autoUpdateTrackers = true;
  DateTime? lastTrackerUpdateTime;

  List<String> customTrackers = [];

  ExtraConfigBt();

  factory ExtraConfigBt.fromJson(Map<String, dynamic> json) =>
      _$ExtraConfigBtFromJson(json);

  Map<String, dynamic> toJson() => _$ExtraConfigBtToJson(this);
}

enum GithubMirrorType {
  jsdelivr,
  ghProxy,
}

@JsonSerializable()
class GithubMirror {
  GithubMirrorType type;
  String url;
  bool isBuiltIn;
  bool isDeleted;

  GithubMirror({
    required this.type,
    required this.url,
    this.isBuiltIn = false,
    this.isDeleted = false,
  });

  factory GithubMirror.fromJson(Map<String, dynamic> json) =>
      _$GithubMirrorFromJson(json);

  Map<String, dynamic> toJson() => _$GithubMirrorToJson(this);
}

@JsonSerializable(explicitToJson: true)
class ExtraConfigGithubMirror {
  bool enabled;
  List<GithubMirror> mirrors;

  ExtraConfigGithubMirror({
    this.enabled = true,
    this.mirrors = const [],
  });

  factory ExtraConfigGithubMirror.fromJson(Map<String, dynamic>? json) =>
      json == null
          ? ExtraConfigGithubMirror()
          : _$ExtraConfigGithubMirrorFromJson(json);

  Map<String, dynamic> toJson() => _$ExtraConfigGithubMirrorToJson(this);
}

@JsonSerializable()
class ArchiveConfig {
  bool autoExtract;
  bool deleteAfterExtract;

  ArchiveConfig({
    this.autoExtract = true,
    this.deleteAfterExtract = true,
  });

  factory ArchiveConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ArchiveConfig() : _$ArchiveConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ArchiveConfigToJson(this);
}
