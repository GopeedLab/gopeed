// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'downloader_config.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

DownloaderConfig _$DownloaderConfigFromJson(Map<String, dynamic> json) =>
    DownloaderConfig(
        downloadDir: json['downloadDir'] as String? ?? '',
        maxRunning: (json['maxRunning'] as num?)?.toInt() ?? 0,
        autoStartTasks: json['autoStartTasks'] as bool? ?? false,
        autoDeleteMissingFileTasks: json['autoDeleteMissingFileTasks'] as bool? ?? false,
      )
      ..protocolConfig = ProtocolConfig.fromJson(json['protocolConfig'] as Map<String, dynamic>?)
      ..extra = ExtraConfig.fromJson(json['extra'] as Map<String, dynamic>?)
      ..proxy = ProxyConfig.fromJson(json['proxy'] as Map<String, dynamic>)
      ..webhook = WebhookConfig.fromJson(json['webhook'] as Map<String, dynamic>?)
      ..script = ScriptConfig.fromJson(json['script'] as Map<String, dynamic>?)
      ..autoTorrent = AutoTorrentConfig.fromJson(json['autoTorrent'] as Map<String, dynamic>?)
      ..archive = ArchiveConfig.fromJson(json['archive'] as Map<String, dynamic>?)
      ..api = ApiServerConfig.fromJson(json['api'] as Map<String, dynamic>?);

Map<String, dynamic> _$DownloaderConfigToJson(DownloaderConfig instance) => <String, dynamic>{
  'downloadDir': instance.downloadDir,
  'maxRunning': instance.maxRunning,
  'protocolConfig': instance.protocolConfig.toJson(),
  'extra': instance.extra.toJson(),
  'proxy': instance.proxy.toJson(),
  'webhook': instance.webhook.toJson(),
  'script': instance.script.toJson(),
  'autoTorrent': instance.autoTorrent.toJson(),
  'archive': instance.archive.toJson(),
  'api': instance.api.toJson(),
  'autoStartTasks': instance.autoStartTasks,
  'autoDeleteMissingFileTasks': instance.autoDeleteMissingFileTasks,
};

ApiServerConfig _$ApiServerConfigFromJson(Map<String, dynamic> json) => ApiServerConfig(
  enable: json['enable'] as bool? ?? false,
  network: json['network'] as String? ?? 'tcp',
  address: json['address'] as String? ?? '127.0.0.1:9999',
  token: json['token'] as String? ?? '',
);

Map<String, dynamic> _$ApiServerConfigToJson(ApiServerConfig instance) => <String, dynamic>{
  'enable': instance.enable,
  'network': instance.network,
  'address': instance.address,
  'token': instance.token,
};

ProtocolConfig _$ProtocolConfigFromJson(Map<String, dynamic> json) => ProtocolConfig()
  ..http = HttpConfig.fromJson(json['http'] as Map<String, dynamic>)
  ..bt = BtConfig.fromJson(json['bt'] as Map<String, dynamic>)
  ..ed2k = Ed2kConfig.fromJson(json['ed2k'] as Map<String, dynamic>);

Map<String, dynamic> _$ProtocolConfigToJson(ProtocolConfig instance) => <String, dynamic>{
  'http': instance.http.toJson(),
  'bt': instance.bt.toJson(),
  'ed2k': instance.ed2k.toJson(),
};

HttpConfig _$HttpConfigFromJson(Map<String, dynamic> json) => HttpConfig(
  userAgent: json['userAgent'] as String? ?? '',
  connections: (json['connections'] as num?)?.toInt() ?? 0,
  useServerCtime: json['useServerCtime'] as bool? ?? false,
);

Map<String, dynamic> _$HttpConfigToJson(HttpConfig instance) => <String, dynamic>{
  'userAgent': instance.userAgent,
  'connections': instance.connections,
  'useServerCtime': instance.useServerCtime,
};

BtConfig _$BtConfigFromJson(Map<String, dynamic> json) => BtConfig(
  listenPort: (json['listenPort'] as num?)?.toInt() ?? 0,
  trackers: (json['trackers'] as List<dynamic>?)?.map((e) => e as String).toList() ?? const [],
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

Ed2kConfig _$Ed2kConfigFromJson(Map<String, dynamic> json) => Ed2kConfig(
  listenPort: (json['listenPort'] as num?)?.toInt() ?? 0,
  udpPort: (json['udpPort'] as num?)?.toInt() ?? 0,
  serverAddr: json['serverAddr'] as String? ?? '',
  serverMet: json['serverMet'] as String? ?? '',
  nodesDat: json['nodesDat'] as String? ?? '',
);

Map<String, dynamic> _$Ed2kConfigToJson(Ed2kConfig instance) => <String, dynamic>{
  'listenPort': instance.listenPort,
  'udpPort': instance.udpPort,
  'serverAddr': instance.serverAddr,
  'serverMet': instance.serverMet,
  'nodesDat': instance.nodesDat,
};

ExtraConfig _$ExtraConfigFromJson(Map<String, dynamic> json) =>
    ExtraConfig(
        themeMode: json['themeMode'] as String? ?? '',
        themeColor: json['themeColor'] as String? ?? 'green',
        locale: json['locale'] as String? ?? '',
        lastDeleteTaskKeep: json['lastDeleteTaskKeep'] as bool? ?? false,
        defaultDirectDownload: json['defaultDirectDownload'] as bool? ?? false,
        defaultBtClient: json['defaultBtClient'] as bool? ?? true,
        notifyWhenNewVersion: json['notifyWhenNewVersion'] as bool? ?? true,
        desktopNotification: json['desktopNotification'] as bool? ?? true,
        backgroundLocationKeepAlive: json['backgroundLocationKeepAlive'] as bool? ?? false,
        windowState: json['windowState'] == null
            ? null
            : WindowStateConfig.fromJson(json['windowState'] as Map<String, dynamic>?),
        bookmarks: (json['bookmarks'] as Map<String, dynamic>?)?.map((k, e) => MapEntry(k, e as String)) ?? const {},
        createHistory: (json['createHistory'] as List<dynamic>?)?.map((e) => e as String).toList() ?? const [],
        runAsMenubarApp: json['runAsMenubarApp'] as bool? ?? false,
        analyticsEnabled: json['analyticsEnabled'] as bool? ?? true,
        analyticsClientId: json['analyticsClientId'] as String? ?? '',
        downloadCategories:
            (json['downloadCategories'] as List<dynamic>?)
                ?.map((e) => DownloadCategory.fromJson(e as Map<String, dynamic>))
                .toList() ??
            const [],
      )
      ..bt = ExtraConfigBt.fromJson(json['bt'] as Map<String, dynamic>)
      ..githubMirror = ExtraConfigGithubMirror.fromJson(json['githubMirror'] as Map<String, dynamic>?);

Map<String, dynamic> _$ExtraConfigToJson(ExtraConfig instance) => <String, dynamic>{
  'themeMode': instance.themeMode,
  'themeColor': instance.themeColor,
  'locale': instance.locale,
  'lastDeleteTaskKeep': instance.lastDeleteTaskKeep,
  'defaultDirectDownload': instance.defaultDirectDownload,
  'defaultBtClient': instance.defaultBtClient,
  'notifyWhenNewVersion': instance.notifyWhenNewVersion,
  'desktopNotification': instance.desktopNotification,
  'backgroundLocationKeepAlive': instance.backgroundLocationKeepAlive,
  'windowState': instance.windowState.toJson(),
  'bookmarks': instance.bookmarks,
  'createHistory': instance.createHistory,
  'runAsMenubarApp': instance.runAsMenubarApp,
  'analyticsEnabled': instance.analyticsEnabled,
  'analyticsClientId': instance.analyticsClientId,
  'downloadCategories': instance.downloadCategories.map((e) => e.toJson()).toList(),
  'bt': instance.bt.toJson(),
  'githubMirror': instance.githubMirror.toJson(),
};

WindowStateConfig _$WindowStateConfigFromJson(Map<String, dynamic> json) => WindowStateConfig(
  isMaximized: json['isMaximized'] as bool? ?? false,
  width: (json['width'] as num?)?.toDouble(),
  height: (json['height'] as num?)?.toDouble(),
);

Map<String, dynamic> _$WindowStateConfigToJson(WindowStateConfig instance) => <String, dynamic>{
  'isMaximized': instance.isMaximized,
  'width': ?instance.width,
  'height': ?instance.height,
};

DownloadCategory _$DownloadCategoryFromJson(Map<String, dynamic> json) => DownloadCategory(
  name: json['name'] as String,
  path: json['path'] as String,
  isBuiltIn: json['isBuiltIn'] as bool? ?? false,
  nameKey: json['nameKey'] as String?,
  isDeleted: json['isDeleted'] as bool? ?? false,
);

Map<String, dynamic> _$DownloadCategoryToJson(DownloadCategory instance) => <String, dynamic>{
  'name': instance.name,
  'path': instance.path,
  'isBuiltIn': instance.isBuiltIn,
  'nameKey': ?instance.nameKey,
  'isDeleted': instance.isDeleted,
};

WebhookConfig _$WebhookConfigFromJson(Map<String, dynamic> json) => WebhookConfig(
  enable: json['enable'] as bool? ?? false,
  urls: (json['urls'] as List<dynamic>?)?.map((e) => e as String).toList() ?? const [],
);

Map<String, dynamic> _$WebhookConfigToJson(WebhookConfig instance) => <String, dynamic>{
  'enable': instance.enable,
  'urls': instance.urls,
};

ScriptConfig _$ScriptConfigFromJson(Map<String, dynamic> json) => ScriptConfig(
  enable: json['enable'] as bool? ?? false,
  paths: (json['paths'] as List<dynamic>?)?.map((e) => e as String).toList() ?? const [],
);

Map<String, dynamic> _$ScriptConfigToJson(ScriptConfig instance) => <String, dynamic>{
  'enable': instance.enable,
  'paths': instance.paths,
};

ProxyConfig _$ProxyConfigFromJson(Map<String, dynamic> json) => ProxyConfig(
  enable: json['enable'] as bool? ?? false,
  system: json['system'] as bool? ?? false,
  scheme: json['scheme'] as String? ?? '',
  host: json['host'] as String? ?? '',
  usr: json['usr'] as String? ?? '',
  pwd: json['pwd'] as String? ?? '',
);

Map<String, dynamic> _$ProxyConfigToJson(ProxyConfig instance) => <String, dynamic>{
  'enable': instance.enable,
  'system': instance.system,
  'scheme': instance.scheme,
  'host': instance.host,
  'usr': instance.usr,
  'pwd': instance.pwd,
};

ExtraConfigBt _$ExtraConfigBtFromJson(Map<String, dynamic> json) => ExtraConfigBt()
  ..trackerSubscribeUrls = (json['trackerSubscribeUrls'] as List<dynamic>).map((e) => e as String).toList()
  ..subscribeTrackers = (json['subscribeTrackers'] as List<dynamic>).map((e) => e as String).toList()
  ..autoUpdateTrackers = json['autoUpdateTrackers'] as bool
  ..lastTrackerUpdateTime = json['lastTrackerUpdateTime'] == null
      ? null
      : DateTime.parse(json['lastTrackerUpdateTime'] as String)
  ..customTrackers = (json['customTrackers'] as List<dynamic>).map((e) => e as String).toList();

Map<String, dynamic> _$ExtraConfigBtToJson(ExtraConfigBt instance) => <String, dynamic>{
  'trackerSubscribeUrls': instance.trackerSubscribeUrls,
  'subscribeTrackers': instance.subscribeTrackers,
  'autoUpdateTrackers': instance.autoUpdateTrackers,
  'lastTrackerUpdateTime': ?instance.lastTrackerUpdateTime?.toIso8601String(),
  'customTrackers': instance.customTrackers,
};

GithubMirror _$GithubMirrorFromJson(Map<String, dynamic> json) => GithubMirror(
  type: $enumDecode(_$GithubMirrorTypeEnumMap, json['type']),
  url: json['url'] as String,
  isBuiltIn: json['isBuiltIn'] as bool? ?? false,
  isDeleted: json['isDeleted'] as bool? ?? false,
);

Map<String, dynamic> _$GithubMirrorToJson(GithubMirror instance) => <String, dynamic>{
  'type': _$GithubMirrorTypeEnumMap[instance.type]!,
  'url': instance.url,
  'isBuiltIn': instance.isBuiltIn,
  'isDeleted': instance.isDeleted,
};

const _$GithubMirrorTypeEnumMap = {GithubMirrorType.jsdelivr: 'jsdelivr', GithubMirrorType.ghProxy: 'ghProxy'};

ExtraConfigGithubMirror _$ExtraConfigGithubMirrorFromJson(Map<String, dynamic> json) => ExtraConfigGithubMirror(
  enabled: json['enabled'] as bool? ?? true,
  mirrors:
      (json['mirrors'] as List<dynamic>?)?.map((e) => GithubMirror.fromJson(e as Map<String, dynamic>)).toList() ??
      const [],
);

Map<String, dynamic> _$ExtraConfigGithubMirrorToJson(ExtraConfigGithubMirror instance) => <String, dynamic>{
  'enabled': instance.enabled,
  'mirrors': instance.mirrors.map((e) => e.toJson()).toList(),
};

AutoTorrentConfig _$AutoTorrentConfigFromJson(Map<String, dynamic> json) => AutoTorrentConfig(
  enable: json['enable'] as bool? ?? false,
  deleteAfterDownload: json['deleteAfterDownload'] as bool? ?? false,
);

Map<String, dynamic> _$AutoTorrentConfigToJson(AutoTorrentConfig instance) => <String, dynamic>{
  'enable': instance.enable,
  'deleteAfterDownload': instance.deleteAfterDownload,
};

ArchiveConfig _$ArchiveConfigFromJson(Map<String, dynamic> json) => ArchiveConfig(
  autoExtract: json['autoExtract'] as bool? ?? true,
  deleteAfterExtract: json['deleteAfterExtract'] as bool? ?? true,
);

Map<String, dynamic> _$ArchiveConfigToJson(ArchiveConfig instance) => <String, dynamic>{
  'autoExtract': instance.autoExtract,
  'deleteAfterExtract': instance.deleteAfterExtract,
};
