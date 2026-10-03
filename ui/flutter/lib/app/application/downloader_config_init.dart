import 'package:path/path.dart' as path;

import '../../api/model/downloader_config.dart';

/// Reads the downloader config, applies first-launch defaults and persists
/// the result when the init changed it. Persisting is skipped when the
/// initial load failed: writing defaults over an unreadable store would
/// wipe the user's configuration.
Future<DownloaderConfig> loadDownloaderConfig({
  required Future<DownloaderConfig> Function() load,
  required Future<void> Function(DownloaderConfig config) save,
  required Future<String> Function() defaultDownloadDir,
  void Function(Object error, StackTrace stackTrace)? onLoadError,
  void Function(Object error, StackTrace stackTrace)? onSaveError,
}) async {
  DownloaderConfig config;
  var loaded = true;
  try {
    config = await load();
  } catch (error, stackTrace) {
    onLoadError?.call(error, stackTrace);
    config = DownloaderConfig();
    loaded = false;
  }
  final changed = await initDownloaderConfig(config, defaultDownloadDir: defaultDownloadDir);
  if (loaded && changed) {
    // Persist the migrated/seeded categories so the backend can route
    // downloads to them right away instead of waiting for the settings page
    // to save the config.
    try {
      await save(config);
    } catch (error, stackTrace) {
      onSaveError?.call(error, stackTrace);
    }
  }
  return config;
}

/// Applies defaults to a freshly loaded config: theme/tracker/proxy basics,
/// migrating the legacy `extra.downloadCategories` list into the typed
/// `categories` field the backend reads for auto categorization, and seeding
/// the built-in categories when nothing exists yet. Returns whether the
/// config changed and should be persisted.
Future<bool> initDownloaderConfig(
  DownloaderConfig config, {
  required Future<String> Function() defaultDownloadDir,
}) async {
  var changed = false;
  final extra = config.extra;
  if (extra.themeMode.isEmpty) {
    extra.themeMode = 'system';
  }
  if (extra.themeColor.isEmpty) {
    extra.themeColor = 'green';
  }
  if (extra.bt.trackerSubscribeUrls.isEmpty) {
    extra.bt.trackerSubscribeUrls.addAll(allTrackerSubscribeUrls);
  }
  if (config.proxy.scheme.isEmpty) {
    config.proxy.scheme = 'http';
  }
  if (config.downloadDir.isEmpty) {
    config.downloadDir = await defaultDownloadDir();
  }
  // Migrate the categories from their legacy extra location to the typed
  // config field the backend reads for auto categorization. Legacy entries
  // predate the extension list, so built-in categories get their default
  // extensions backfilled; user created categories keep their empty list
  // until the user edits them.
  if (config.categories.isEmpty && extra.downloadCategories.isNotEmpty) {
    config.categories = List<DownloadCategory>.from(extra.downloadCategories);
    for (final category in config.categories) {
      if (category.nameKey != null && category.extensions.isEmpty) {
        category.extensions = List<String>.from(kDefaultCategoryExtensions[category.nameKey] ?? const []);
      }
    }
    changed = true;
  }
  if (extra.downloadCategories.isNotEmpty) {
    extra.downloadCategories = [];
    changed = true;
  }
  if (config.categories.isEmpty) {
    config.categories = [
      _builtinCategory('categoryMusic', path.join(config.downloadDir, 'Music')),
      _builtinCategory('categoryVideo', path.join(config.downloadDir, 'Video')),
      _builtinCategory('categoryDocument', path.join(config.downloadDir, 'Document')),
      _builtinCategory('categoryProgram', path.join(config.downloadDir, 'Program')),
    ];
    changed = true;
  }
  if (extra.githubMirror.mirrors.isEmpty) {
    extra.githubMirror.mirrors = [
      GithubMirror(type: GithubMirrorType.jsdelivr, url: 'https://fastly.jsdelivr.net/gh', isBuiltIn: true),
      GithubMirror(type: GithubMirrorType.ghProxy, url: 'https://fastgit.cc', isBuiltIn: true),
    ];
  }
  return changed;
}

DownloadCategory _builtinCategory(String nameKey, String categoryPath) {
  return DownloadCategory(
    name: '',
    path: categoryPath,
    isBuiltIn: true,
    nameKey: nameKey,
    extensions: List<String>.from(kDefaultCategoryExtensions[nameKey] ?? const []),
  );
}
