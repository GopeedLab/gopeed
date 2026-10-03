import 'package:json_annotation/json_annotation.dart';
import 'package:path/path.dart' as path;

part 'downloader_config.g.dart';

@JsonSerializable(explicitToJson: true)
class DownloaderConfig {
  String downloadDir;
  int maxRunning;
  ProtocolConfig protocolConfig = ProtocolConfig();
  ExtraConfig extra = ExtraConfig();
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
  // Legacy location of the download categories, migrated to
  // DownloaderConfig.categories on startup. Kept so old configurations can be
  // read, and cleared right after the migration ran.
  List<DownloadCategory> downloadCategories;

  ExtraConfigBt bt = ExtraConfigBt();
  ExtraConfigGithubMirror githubMirror = ExtraConfigGithubMirror();

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
    this.downloadCategories = const [],
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

/// Normalizes free form extension input into tokens: splits on commas and
/// whitespace, trims, lowercases, strips leading dots, drops empties and
/// deduplicates while keeping the original input order.
List<String> normalizeCategoryExtensions(Iterable<String> raw) {
  final result = <String>{};
  for (final item in raw) {
    for (final part in item.split(RegExp(r'[,\s]+'))) {
      var extension = part.trim().toLowerCase();
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

/// Default tracker list urls for the bt subscribe setting. Kept in the
/// model layer so the config init helpers (downloader_config_init.dart)
/// and the settings page can share it without importing the riverpod
/// controller.
const allTrackerSubscribeUrls = [
  'https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt',
  'https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_http.txt',
  'https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_https.txt',
  'https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_ip.txt',
  'https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_udp.txt',
  'https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all_ws.txt',
  'https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt',
  'https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best_ip.txt',
  'https://raw.githubusercontent.com/XIU2/TrackersListCollection/master/all.txt',
  'https://raw.githubusercontent.com/XIU2/TrackersListCollection/master/best.txt',
  'https://raw.githubusercontent.com/XIU2/TrackersListCollection/master/http.txt',
];

/// Lower case extension of [name] without the leading dot, e.g. "setup.exe" -> "exe".
/// Dots inside directory segments are ignored, mirroring the backend so a
/// rename like "dir.d/file" never counts as a candidate.
String categoryFileExtension(String name) {
  final cleaned = name.replaceAll(r'\', '/');
  final slash = cleaned.lastIndexOf('/');
  final index = cleaned.lastIndexOf('.');
  if (index <= slash || index == cleaned.length - 1) {
    return '';
  }
  return cleaned.substring(index + 1).toLowerCase();
}

/// Derives a display file name from a task url for category matching,
/// ignoring the query and fragment part. Returns an empty string when the
/// url carries no usable name.
String categoryFileNameFromUrl(String rawUrl) {
  var cleaned = rawUrl;
  final queryIndex = cleaned.indexOf('?');
  final fragmentIndex = cleaned.indexOf('#');
  final cutIndex = switch ((queryIndex, fragmentIndex)) {
    (< 0, < 0) => -1,
    (>= 0, < 0) => queryIndex,
    (< 0, >= 0) => fragmentIndex,
    _ => queryIndex < fragmentIndex ? queryIndex : fragmentIndex,
  };
  if (cutIndex >= 0) {
    cleaned = cleaned.substring(0, cutIndex);
  }
  final lower = cleaned.toLowerCase();
  if (lower.startsWith('ed2k:')) {
    // ed2k://|file|name|size|hash|/
    final parts = cleaned.split('|');
    if (parts.length >= 4 && parts[1].toLowerCase() == 'file') {
      return parts[2].trim();
    }
    return '';
  }
  if (lower.startsWith('data:') || lower.startsWith('blob:') || lower.startsWith('magnet:')) {
    return '';
  }
  final lastSlash = cleaned.lastIndexOf('/');
  final lastBackslash = cleaned.lastIndexOf(r'\');
  final lastSeparator = lastSlash > lastBackslash ? lastSlash : lastBackslash;
  final segment = (lastSeparator >= 0 ? cleaned.substring(lastSeparator + 1) : cleaned).trim();
  try {
    // Backend url parsing percent-decodes the path, keep the hint in sync.
    return Uri.decodeComponent(segment);
  } catch (_) {
    // Malformed escapes fall back to the raw segment.
    return segment;
  }
}

/// The best available file name for category matching: the first candidate
/// that carries an extension, mirroring the backend candidate order so the
/// create page hint never disagrees with the real routing.
String categoryCandidateFileName({String? rename, String? url}) {
  final name = rename?.trim() ?? '';
  if (name.isNotEmpty && categoryFileExtension(name) != '') {
    return name;
  }
  final urlName = categoryFileNameFromUrl(url?.trim() ?? '');
  if (categoryFileExtension(urlName) != '') {
    return urlName;
  }
  return '';
}

/// Returns the category that should receive [fileName], or null when nothing
/// matches. Deleted categories and categories without a path are skipped.
DownloadCategory? matchDownloadCategory(List<DownloadCategory> categories, String fileName) {
  final ext = categoryFileExtension(fileName);
  if (ext.isEmpty) {
    return null;
  }
  for (final category in categories) {
    if (category.isDeleted || category.path.trim().isEmpty) {
      continue;
    }
    // effectiveExtensions already normalizes the stored entries.
    for (final candidate in category.effectiveExtensions()) {
      if (candidate == ext) {
        return category;
      }
    }
  }
  return null;
}

/// Rebases built-in category paths that still live under [oldDir] onto
/// [newDir] after the default download directory changed. Categories the
/// user moved elsewhere, and user created categories, are left alone.
void rebaseBuiltinCategoryPaths(DownloaderConfig config, {required String oldDir, required String newDir}) {
  if (oldDir.trim().isEmpty || newDir.trim().isEmpty || path.equals(oldDir, newDir)) {
    return;
  }
  for (final category in config.categories) {
    if (!category.isBuiltIn || category.isDeleted || category.path.trim().isEmpty) {
      continue;
    }
    if (path.equals(category.path, oldDir)) {
      category.path = newDir;
    } else if (path.isWithin(oldDir, category.path)) {
      category.path = path.join(newDir, path.relative(category.path, from: oldDir));
    }
  }
}

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

@JsonSerializable()
class ExtraConfigBt {
  List<String> trackerSubscribeUrls = [];
  List<String> subscribeTrackers = [];
  bool autoUpdateTrackers = true;
  DateTime? lastTrackerUpdateTime;

  List<String> customTrackers = [];

  ExtraConfigBt();

  factory ExtraConfigBt.fromJson(Map<String, dynamic> json) => _$ExtraConfigBtFromJson(json);

  Map<String, dynamic> toJson() => _$ExtraConfigBtToJson(this);
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
class ExtraConfigGithubMirror {
  bool enabled;
  List<GithubMirror> mirrors;

  ExtraConfigGithubMirror({this.enabled = true, this.mirrors = const []});

  factory ExtraConfigGithubMirror.fromJson(Map<String, dynamic>? json) =>
      json == null ? ExtraConfigGithubMirror() : _$ExtraConfigGithubMirrorFromJson(json);

  Map<String, dynamic> toJson() => _$ExtraConfigGithubMirrorToJson(this);
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
