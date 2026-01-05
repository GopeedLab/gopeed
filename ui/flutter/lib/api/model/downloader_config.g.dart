// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'downloader_config.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DownloaderConfig _$DownloaderConfigFromJson(Map<String, dynamic> json) =>
    DownloaderConfig(
      downloadDir: json['downloadDir'] as String? ?? '',
      maxRunning: (json['maxRunning'] as num?)?.toInt() ?? 0,
    )
      ..protocolConfig = ProtocolConfig.fromJson(
          json['protocolConfig'] as Map<String, dynamic>?)
      ..extra = ExtraConfig.fromJson(json['extra'] as Map<String, dynamic>?)
      ..proxy = ProxyConfig.fromJson(json['proxy'] as Map<String, dynamic>)
      ..webhook =
          WebhookConfig.fromJson(json['webhook'] as Map<String, dynamic>?)
      ..archive =
          ArchiveConfig.fromJson(json['archive'] as Map<String, dynamic>?);

Map<String, dynamic> _$DownloaderConfigToJson(DownloaderConfig instance) =>
    <String, dynamic>{
      'downloadDir': instance.downloadDir,
      'maxRunning': instance.maxRunning,
      'protocolConfig': instance.protocolConfig.toJson(),
      'extra': instance.extra.toJson(),
      'proxy': instance.proxy.toJson(),
      'webhook': instance.webhook.toJson(),
      'archive': instance.archive.toJson(),
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

HttpConfig _$HttpConfigFromJson(Map<String, dynamic> json) => HttpConfig(
      userAgent: json['userAgent'] as String? ?? '',
      connections: (json['connections'] as num?)?.toInt() ?? 0,
      useServerCtime: json['useServerCtime'] as bool? ?? false,
    );

Map<String, dynamic> _$HttpConfigToJson(HttpConfig instance) =>
    <String, dynamic>{
      'userAgent': instance.userAgent,
      'connections': instance.connections,
      'useServerCtime': instance.useServerCtime,
    };

BtConfig _$BtConfigFromJson(Map<String, dynamic> json) => BtConfig(
      listenPort: (json['listenPort'] as num?)?.toInt() ?? 0,
      trackers: (json['trackers'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          const [],
      seedKeep: json['seedKeep'] as bool? ?? false,
      seedRatio: (json['seedRatio'] as num?)?.toDouble() ?? 0,
      seedTime: (json['seedTime'] as num?)?.toInt() ?? 0,
    );

Map<String, dynamic> _$BtConfigToJson(BtConfig instance) => <String, dynamic>{
      'listenPort': instance.listenPort,
      'trackers': instance.trackers,
      'seedKeep': instance.seedKeep,
      'seedRatio': instance.seedRatio,
      'seedTime': instance.seedTime,
    };

ExtraConfig _$ExtraConfigFromJson(Map<String, dynamic> json) => ExtraConfig(
      themeMode: json['themeMode'] as String? ?? '',
      locale: json['locale'] as String? ?? '',
      lastDeleteTaskKeep: json['lastDeleteTaskKeep'] as bool? ?? false,
      defaultDirectDownload: json['defaultDirectDownload'] as bool? ?? false,
      defaultBtClient: json['defaultBtClient'] as bool? ?? true,
      notifyWhenNewVersion: json['notifyWhenNewVersion'] as bool? ?? true,
      downloadCategories: (json['downloadCategories'] as List<dynamic>?)
              ?.map((e) => DownloadCategory.fromJson(e as Map<String, dynamic>))
              .toList() ??
          const [],
    )
      ..bt = ExtraConfigBt.fromJson(json['bt'] as Map<String, dynamic>)
      ..githubMirror = ExtraConfigGithubMirror.fromJson(
          json['githubMirror'] as Map<String, dynamic>?);

Map<String, dynamic> _$ExtraConfigToJson(ExtraConfig instance) =>
    <String, dynamic>{
      'themeMode': instance.themeMode,
      'locale': instance.locale,
      'lastDeleteTaskKeep': instance.lastDeleteTaskKeep,
      'defaultDirectDownload': instance.defaultDirectDownload,
      'defaultBtClient': instance.defaultBtClient,
      'notifyWhenNewVersion': instance.notifyWhenNewVersion,
      'downloadCategories':
          instance.downloadCategories.map((e) => e.toJson()).toList(),
      'bt': instance.bt.toJson(),
      'githubMirror': instance.githubMirror.toJson(),
    };

DownloadCategory _$DownloadCategoryFromJson(Map<String, dynamic> json) =>
    DownloadCategory(
      name: json['name'] as String,
      path: json['path'] as String,
      isBuiltIn: json['isBuiltIn'] as bool? ?? false,
      nameKey: json['nameKey'] as String?,
      isDeleted: json['isDeleted'] as bool? ?? false,
    );

Map<String, dynamic> _$DownloadCategoryToJson(DownloadCategory instance) {
  final val = <String, dynamic>{
    'name': instance.name,
    'path': instance.path,
    'isBuiltIn': instance.isBuiltIn,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('nameKey', instance.nameKey);
  val['isDeleted'] = instance.isDeleted;
  return val;
}

WebhookConfig _$WebhookConfigFromJson(Map<String, dynamic> json) =>
    WebhookConfig(
      enable: json['enable'] as bool? ?? false,
      urls:
          (json['urls'] as List<dynamic>?)?.map((e) => e as String).toList() ??
              const [],
    );

Map<String, dynamic> _$WebhookConfigToJson(WebhookConfig instance) =>
    <String, dynamic>{
      'enable': instance.enable,
      'urls': instance.urls,
    };

ProxyConfig _$ProxyConfigFromJson(Map<String, dynamic> json) => ProxyConfig(
      enable: json['enable'] as bool? ?? false,
      system: json['system'] as bool? ?? false,
      scheme: json['scheme'] as String? ?? '',
      host: json['host'] as String? ?? '',
      usr: json['usr'] as String? ?? '',
      pwd: json['pwd'] as String? ?? '',
    );

Map<String, dynamic> _$ProxyConfigToJson(ProxyConfig instance) =>
    <String, dynamic>{
      'enable': instance.enable,
      'system': instance.system,
      'scheme': instance.scheme,
      'host': instance.host,
      'usr': instance.usr,
      'pwd': instance.pwd,
    };

ExtraConfigBt _$ExtraConfigBtFromJson(Map<String, dynamic> json) =>
    ExtraConfigBt()
      ..trackerSubscribeUrls = (json['trackerSubscribeUrls'] as List<dynamic>)
          .map((e) => e as String)
          .toList()
      ..subscribeTrackers = (json['subscribeTrackers'] as List<dynamic>)
          .map((e) => e as String)
          .toList()
      ..autoUpdateTrackers = json['autoUpdateTrackers'] as bool
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
    'autoUpdateTrackers': instance.autoUpdateTrackers,
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

GithubMirror _$GithubMirrorFromJson(Map<String, dynamic> json) => GithubMirror(
      type: $enumDecode(_$GithubMirrorTypeEnumMap, json['type']),
      url: json['url'] as String,
      isBuiltIn: json['isBuiltIn'] as bool? ?? false,
      isDeleted: json['isDeleted'] as bool? ?? false,
    );

Map<String, dynamic> _$GithubMirrorToJson(GithubMirror instance) =>
    <String, dynamic>{
      'type': _$GithubMirrorTypeEnumMap[instance.type]!,
      'url': instance.url,
      'isBuiltIn': instance.isBuiltIn,
      'isDeleted': instance.isDeleted,
    };

const _$GithubMirrorTypeEnumMap = {
  GithubMirrorType.jsdelivr: 'jsdelivr',
  GithubMirrorType.ghProxy: 'ghProxy',
};

ExtraConfigGithubMirror _$ExtraConfigGithubMirrorFromJson(
        Map<String, dynamic> json) =>
    ExtraConfigGithubMirror(
      enabled: json['enabled'] as bool? ?? true,
      mirrors: (json['mirrors'] as List<dynamic>?)
              ?.map((e) => GithubMirror.fromJson(e as Map<String, dynamic>))
              .toList() ??
          const [],
    );

Map<String, dynamic> _$ExtraConfigGithubMirrorToJson(
        ExtraConfigGithubMirror instance) =>
    <String, dynamic>{
      'enabled': instance.enabled,
      'mirrors': instance.mirrors.map((e) => e.toJson()).toList(),
    };

ArchiveConfig _$ArchiveConfigFromJson(Map<String, dynamic> json) =>
    ArchiveConfig(
      autoExtract: json['autoExtract'] as bool? ?? true,
      deleteAfterExtract: json['deleteAfterExtract'] as bool? ?? true,
    );

Map<String, dynamic> _$ArchiveConfigToJson(ArchiveConfig instance) =>
    <String, dynamic>{
      'autoExtract': instance.autoExtract,
      'deleteAfterExtract': instance.deleteAfterExtract,
    };
