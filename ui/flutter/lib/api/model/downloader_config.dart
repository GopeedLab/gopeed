import 'package:json_annotation/json_annotation.dart';
import 'package:path/path.dart' as path;

part 'downloader_config.g.dart';

@JsonSerializable(explicitToJson: true)
class DownloaderConfig {
  String downloadDir;
  int maxRunning;
  ProtocolConfig protocolConfig = ProtocolConfig();
  ExtraConfig extra = ExtraConfig();
  GithubMirrorConfig githubMirror = GithubMirrorConfig();
  ProxyConfig proxy = ProxyConfig();
  WebhookConfig webhook = WebhookConfig();
  ScriptConfig script = ScriptConfig();
  AutoTorrentConfig autoTorrent = AutoTorrentConfig();
  ArchiveConfig archive = ArchiveConfig();
  ApiServerConfig api = ApiServerConfig();
  bool autoStartTasks;
  bool autoDeleteMissingFileTasks;
  List<DownloadCategory> categories;
  bool autoCategorize;

  DownloaderConfig({
    this.downloadDir = '',
    this.maxRunning = 0,
    this.autoStartTasks = false,
    this.autoDeleteMissingFileTasks = false,
    this.categories = const [],
    this.autoCategorize = false,
  });

  factory DownloaderConfig.fromJson(Map<String, dynamic> json) => _$DownloaderConfigFromJson(json);

  Map<String, dynamic> toJson() => _$DownloaderConfigToJson(this);
}

@JsonSerializable()
class ApiServerConfig {
  bool enable;
  bool mcpEnable;
  String network;
  String address;
  String token;

  ApiServerConfig({
    this.enable = false,
    this.mcpEnable = false,
    this.network = 'tcp',
    this.address = '127.0.0.1:9999',
    this.token = '',
  });

  factory ApiServerConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ApiServerConfig() : _$ApiServerConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ApiServerConfigToJson(this);
}

@JsonSerializable(explicitToJson: true)
class ProtocolConfig {
  HttpConfig http = HttpConfig();
  BtConfig bt = BtConfig();
  Ed2kConfig ed2k = Ed2kConfig();
  HlsConfig hls = HlsConfig();

  ProtocolConfig();

  factory ProtocolConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ProtocolConfig() : _$ProtocolConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ProtocolConfigToJson(this);
}

@JsonSerializable()
class HlsConfig {
  int segmentConnections;
  int maxRetries;
  int timeoutSeconds;
  bool prefetchContentLength;

  HlsConfig({
    this.segmentConnections = 0,
    this.maxRetries = 0,
    this.timeoutSeconds = 0,
    this.prefetchContentLength = false,
  });

  factory HlsConfig.fromJson(Map<String, dynamic> json) => _$HlsConfigFromJson(json);

  Map<String, dynamic> toJson() => _$HlsConfigToJson(this);
}

@JsonSerializable()
class HttpConfig {
  String userAgent;
  int connections;
  bool useServerCtime;

  HttpConfig({this.userAgent = '', this.connections = 0, this.useServerCtime = false});

  factory HttpConfig.fromJson(Map<String, dynamic> json) => _$HttpConfigFromJson(json);

  Map<String, dynamic> toJson() => _$HttpConfigToJson(this);
}

DateTime? _trackerUpdateTimeFromJson(String? value) =>
    value == null || value.startsWith('0001-') ? null : DateTime.parse(value);

@JsonSerializable()
class TrackerSubscriptionStatus {
  final bool success;
  final int trackerCount;
  final DateTime lastFetchTime;

  TrackerSubscriptionStatus({required this.success, required this.trackerCount, required this.lastFetchTime});

  factory TrackerSubscriptionStatus.fromJson(Map<String, dynamic> json) => _$TrackerSubscriptionStatusFromJson(json);

  Map<String, dynamic> toJson() => _$TrackerSubscriptionStatusToJson(this);
}

@JsonSerializable(explicitToJson: true)
class BtState {
  Map<String, List<String>> trackerSources;
  Map<String, TrackerSubscriptionStatus> trackerSubscriptionStatuses;
  @JsonKey(fromJson: _trackerUpdateTimeFromJson)
  DateTime? lastTrackerUpdateTime;

  BtState({this.trackerSources = const {}, this.trackerSubscriptionStatuses = const {}, this.lastTrackerUpdateTime});

  factory BtState.fromJson(Map<String, dynamic>? json) => json == null ? BtState() : _$BtStateFromJson(json);

  Map<String, dynamic> toJson() => _$BtStateToJson(this);
}

@JsonSerializable(explicitToJson: true)
class BtConfig {
  BtState state = BtState();
  List<String> trackerSubscribeUrls;
  bool autoUpdateTrackers;
  List<String> customTrackers;
  int listenPort;
  bool seedKeep;
  double seedRatio;
  int seedTime;
  bool disableDht;
  bool disablePex;

  BtConfig({
    this.trackerSubscribeUrls = const [],
    this.autoUpdateTrackers = true,
    this.customTrackers = const [],
    this.listenPort = 0,
    this.seedKeep = false,
    this.seedRatio = 0,
    this.seedTime = 0,
    this.disableDht = false,
    this.disablePex = false,
  });

  factory BtConfig.fromJson(Map<String, dynamic> json) => _$BtConfigFromJson(json);

  Map<String, dynamic> toJson() => _$BtConfigToJson(this);
}

@JsonSerializable()
class Ed2kConfig {
  int listenPort;
  int udpPort;
  String serverAddr;
  String serverMet;
  String nodesDat;

  Ed2kConfig({this.listenPort = 0, this.udpPort = 0, this.serverAddr = '', this.serverMet = '', this.nodesDat = ''});

  factory Ed2kConfig.fromJson(Map<String, dynamic> json) => _$Ed2kConfigFromJson(json);

  Map<String, dynamic> toJson() => _$Ed2kConfigToJson(this);
}

@JsonSerializable(explicitToJson: true)
class ExtraConfig {
  String themeMode;
  String themeColor;
  String locale;
  bool lastDeleteTaskKeep;
  bool defaultDirectDownload;
  bool defaultBtClient;
  bool notifyWhenNewVersion;
  bool desktopNotification;
  bool backgroundLocationKeepAlive;
  bool backgroundContinuedProcessing;
  WindowStateConfig windowState;
  Map<String, String> bookmarks;
  List<String> createHistory;
  bool runAsMenubarApp;
  bool analyticsEnabled;
  String analyticsClientId;

  ExtraConfig({
    this.themeMode = '',
    this.themeColor = 'green',
    this.locale = '',
    this.lastDeleteTaskKeep = false,
    this.defaultDirectDownload = false,
    this.defaultBtClient = true,
    this.notifyWhenNewVersion = true,
    this.desktopNotification = true,
    this.backgroundLocationKeepAlive = false,
    this.backgroundContinuedProcessing = false,
    WindowStateConfig? windowState,
    this.bookmarks = const {},
    this.createHistory = const [],
    this.runAsMenubarApp = false,
    this.analyticsEnabled = true,
    this.analyticsClientId = '',
  }) : windowState = windowState ?? WindowStateConfig();

  factory ExtraConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ExtraConfig() : _$ExtraConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ExtraConfigToJson(this);
}

@JsonSerializable()
class WindowStateConfig {
  bool isMaximized;
  double? width;
  double? height;

  WindowStateConfig({this.isMaximized = false, this.width, this.height});

  factory WindowStateConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? WindowStateConfig() : _$WindowStateConfigFromJson(json);

  Map<String, dynamic> toJson() => _$WindowStateConfigToJson(this);
}

@JsonSerializable()
class DownloadCategory {
  String name;
  String path;
  bool isBuiltIn;
  String? nameKey; // i18n key for built-in categories (e.g., 'categoryMusic')
  bool isDeleted; // Mark built-in categories as deleted instead of removing them
  // File extensions routed to this category, without leading dots, e.g. ['exe', 'msi'].
  // An empty list means the category never matches automatically.
  List<String> extensions;

  DownloadCategory({
    required this.name,
    required this.path,
    this.isBuiltIn = false,
    this.nameKey,
    this.isDeleted = false,
    this.extensions = const [],
  });

  factory DownloadCategory.fromJson(Map<String, dynamic> json) => _$DownloadCategoryFromJson(json);

  Map<String, dynamic> toJson() => _$DownloadCategoryToJson(this);

  /// The effective extension list. The result is always normalized: split on
  /// commas/whitespace, trimmed, lowercased, without leading dots and without
  /// duplicates; the original order is preserved. An empty list means the
  /// category never matches automatically.
  List<String> effectiveExtensions() => normalizeCategoryExtensions(extensions);
}

/// Lowercases the ascii letters of [value] and leaves the rest alone. The
/// backend lowercases the same way, because the unicode case tables of the two
/// languages disagree for some code points and would let the two sides pick a
/// different category for the same file.
String asciiLower(String value) {
  final lowered = StringBuffer();
  for (final unit in value.codeUnits) {
    lowered.writeCharCode(unit >= 0x41 && unit <= 0x5A ? unit + 0x20 : unit);
  }
  return lowered.toString();
}

/// Normalizes free form extension input into tokens: splits on commas and
/// whitespace, trims, lowercases, strips leading dots, drops empties and
/// deduplicates while keeping the original input order.
List<String> normalizeCategoryExtensions(Iterable<String> raw) {
  final result = <String>{};
  for (final item in raw) {
    for (final part in item.split(RegExp(r'[,\s]+'))) {
      var extension = asciiLower(part.trim());
      while (extension.startsWith('.')) {
        extension = extension.substring(1);
      }
      if (extension.isNotEmpty) {
        result.add(extension);
      }
    }
  }
  return result.toList();
}

/// Default extension lists of the built-in categories, keyed by the i18n
/// nameKey. Used to seed the built-in categories on first launch and to offer
/// the restore-defaults action in the category dialog.
const kDefaultCategoryExtensions = <String, List<String>>{
  'categoryProgram': ['exe', 'msi', 'msix', 'apk', 'dmg', 'deb', 'rpm', 'pkg', 'appimage'],
  'categoryVideo': ['mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm', 'm4v', 'mpg', 'mpeg', '3gp'],
  'categoryMusic': ['mp3', 'flac', 'wav', 'aac', 'ogg', 'm4a', 'wma', 'ape'],
  'categoryDocument': ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'csv', 'epub', 'rtf'],
};

/// Rebases built-in category paths that still live under [oldDir] onto
/// [newDir] after the default download directory changed. Categories the
/// user moved elsewhere, and user created categories, are left alone.
void rebaseBuiltinCategoryPaths(DownloaderConfig config, {required String oldDir, required String newDir}) {
  if (oldDir.trim().isEmpty || newDir.trim().isEmpty) {
    return;
  }
  // The backend seeds the categories with the separator of its platform, which
  // the path package of the client does not always understand, so both sides
  // are compared in the forward slash form.
  final from = _forwardSlash(oldDir);
  final to = _forwardSlash(newDir);
  if (path.equals(from, to)) {
    return;
  }
  for (final category in config.categories) {
    if (!category.isBuiltIn || category.isDeleted || category.path.trim().isEmpty) {
      continue;
    }
    final current = _forwardSlash(category.path);
    if (path.equals(current, from)) {
      category.path = to;
    } else if (path.isWithin(from, current)) {
      category.path = path.join(to, path.relative(current, from: from));
    }
  }
}

/// Rewrites the windows separator of [value] to the forward slash the path
/// package uses.
String _forwardSlash(String value) => value.replaceAll('\\', '/');

@JsonSerializable()
class WebhookConfig {
  bool enable;
  List<String> urls;

  WebhookConfig({this.enable = false, this.urls = const []});

  factory WebhookConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? WebhookConfig() : _$WebhookConfigFromJson(json);

  Map<String, dynamic> toJson() => _$WebhookConfigToJson(this);
}

@JsonSerializable()
class ScriptConfig {
  bool enable;
  List<String> paths;

  ScriptConfig({this.enable = false, this.paths = const []});

  factory ScriptConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ScriptConfig() : _$ScriptConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ScriptConfigToJson(this);
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

  factory ProxyConfig.fromJson(Map<String, dynamic> json) => _$ProxyConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ProxyConfigToJson(this);
}

enum GithubMirrorType { jsdelivr, ghProxy }

@JsonSerializable()
class GithubMirror {
  GithubMirrorType type;
  String url;
  bool isBuiltIn;
  bool isDeleted;

  GithubMirror({required this.type, required this.url, this.isBuiltIn = false, this.isDeleted = false});

  factory GithubMirror.fromJson(Map<String, dynamic> json) => _$GithubMirrorFromJson(json);

  Map<String, dynamic> toJson() => _$GithubMirrorToJson(this);
}

@JsonSerializable(explicitToJson: true)
class GithubMirrorConfig {
  bool enabled;
  List<GithubMirror> mirrors;

  GithubMirrorConfig({this.enabled = true, this.mirrors = const []});

  factory GithubMirrorConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? GithubMirrorConfig() : _$GithubMirrorConfigFromJson(json);

  Map<String, dynamic> toJson() => _$GithubMirrorConfigToJson(this);
}

@JsonSerializable()
class AutoTorrentConfig {
  bool enable;
  bool deleteAfterDownload;

  AutoTorrentConfig({this.enable = false, this.deleteAfterDownload = false});

  factory AutoTorrentConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? AutoTorrentConfig() : _$AutoTorrentConfigFromJson(json);

  Map<String, dynamic> toJson() => _$AutoTorrentConfigToJson(this);
}

@JsonSerializable()
class ArchiveConfig {
  bool autoExtract;
  bool deleteAfterExtract;

  ArchiveConfig({this.autoExtract = true, this.deleteAfterExtract = true});

  factory ArchiveConfig.fromJson(Map<String, dynamic>? json) =>
      json == null ? ArchiveConfig() : _$ArchiveConfigFromJson(json);

  Map<String, dynamic> toJson() => _$ArchiveConfigToJson(this);
}
