import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';

import '../../api/api.dart' as api;
import '../../api/model/downloader_config.dart';
import '../../core/capabilities/app_capabilities.dart';
import '../../core/common/start_config.dart';
import '../../core/entry/app_initializer.dart';
import '../../core/libgopeed_boot.dart';
import '../../database/database.dart';
import '../../database/entity.dart';
import '../../features/auth/application/web_auth_controller.dart';
import '../../l10n/l10n.dart';
import '../../util/log_util.dart';
import '../../util/package_info.dart';
import '../../util/util.dart';
import '../rpc/webview_rpc_service.dart';
import 'android_foreground_service.dart';
import 'location_keep_alive.dart';

const unixSocketPath = 'gopeed.sock';

final appRuntimeControllerProvider = AsyncNotifierProvider<AppRuntimeController, AppRuntimeState>(
  AppRuntimeController.new,
);

class AppRuntimeState {
  const AppRuntimeState({
    required this.startConfig,
    required this.runningPort,
    required this.downloaderConfig,
    this.localBackendStarted = false,
    this.startupError,
  });

  final StartConfig startConfig;
  final int runningPort;
  final DownloaderConfig downloaderConfig;
  final bool localBackendStarted;
  final Object? startupError;

  String runningAddress() {
    if (startConfig.network == 'unix') {
      return startConfig.address;
    }
    return '${startConfig.address.split(':').first}:$runningPort';
  }
}

class AppRuntimeController extends AsyncNotifier<AppRuntimeState> {
  static StartConfig? _defaultStartConfig;

  @override
  Future<AppRuntimeState> build() async {
    ref.onDispose(() => unawaited(WebViewRpcService.instance.stop()));
    return _init();
  }

  Future<void> reloadConfig() async {
    final current = state.value;
    if (current == null) return;
    state = AsyncValue.data(
      AppRuntimeState(
        startConfig: current.startConfig,
        runningPort: current.runningPort,
        downloaderConfig: await ref.read(gopeedServiceProvider).getConfig(),
        localBackendStarted: current.localBackendStarted,
        startupError: current.startupError,
      ),
    );
  }

  void replaceDownloaderConfig(DownloaderConfig config) {
    final current = state.value;
    if (current == null) return;
    state = AsyncValue.data(
      AppRuntimeState(
        startConfig: current.startConfig,
        runningPort: current.runningPort,
        downloaderConfig: DownloaderConfig.fromJson(config.toJson()),
        localBackendStarted: current.localBackendStarted,
        startupError: current.startupError,
      ),
    );
    unawaited(LocationKeepAliveCoordinator.instance.reconcile(enabled: config.extra.backgroundLocationKeepAlive));
  }

  Future<AppRuntimeState> _init() async {
    await AppInitializer.ensureDatabaseInitialized();
    try {
      await initPackageInfo();
    } catch (_) {}
    initLogger();

    final startConfig = await _loadStartConfig();
    try {
      startConfig.webViewRpcConfig = await WebViewRpcService.instance.start();
    } catch (error, stackTrace) {
      logger.w('webview RPC initialization failed', error, stackTrace);
    }
    Object? startupError;
    var runningPort = 0;
    var localBackendStarted = false;
    try {
      // Go's rest.Start is already idempotent: when its server is running it
      // returns the existing port before rebuilding or touching the Unix
      // socket. This is also what keeps Riverpod hot reload safe.
      runningPort = await LibgopeedBoot.instance.start(startConfig);
      localBackendStarted = true;
    } catch (error, stackTrace) {
      startupError = error;
      logger.w('libgopeed init failed, falling back to external API', error, stackTrace);
      if (startConfig.network == 'unix') {
        startConfig.network = 'tcp';
        startConfig.address = '127.0.0.1:9999';
      }
      runningPort = int.tryParse(startConfig.address.split(':').last) ?? 9999;
    }

    api.init(
      startConfig.network,
      _runningAddress(startConfig, runningPort),
      startConfig.apiToken,
      webTokenProvider: Database.instance.getWebToken,
      onUnauthorized: ref.read(webAuthControllerProvider).requireLogin,
    );

    final config = await _loadDownloaderConfig();
    unawaited(_initTrackerUpdate(config));
    unawaited(_initAndroidForegroundService(appLocalizationsFor(config.extra.locale)));

    final result = AppRuntimeState(
      startConfig: startConfig,
      runningPort: runningPort,
      downloaderConfig: config,
      localBackendStarted: localBackendStarted,
      startupError: startupError,
    );
    if (Util.isIOS()) {
      LocationKeepAliveCoordinator.instance.start(
        () =>
            state.value?.downloaderConfig.extra.backgroundLocationKeepAlive ?? config.extra.backgroundLocationKeepAlive,
      );
    }
    return result;
  }

  Future<void> _initAndroidForegroundService(AppLocalizations l10n) async {
    try {
      await AndroidForegroundService.ensureRunning(l10n);
    } catch (error, stackTrace) {
      logger.w('Android foreground service initialization failed', error, stackTrace);
    }
  }

  Future<StartConfig> _loadStartConfig() async {
    final defaultCfg = await _initDefaultStartConfig();
    final saved = Database.instance.getStartConfig();
    final cfg = StartConfig()
      ..network = saved?.network ?? defaultCfg.network
      ..address = saved?.address ?? defaultCfg.address
      ..apiToken = saved?.apiToken ?? defaultCfg.apiToken
      ..storage = defaultCfg.storage
      ..storageDir = defaultCfg.storageDir
      ..refreshInterval = defaultCfg.refreshInterval;
    return cfg;
  }

  Future<StartConfig> _initDefaultStartConfig() async {
    if (_defaultStartConfig != null) {
      return _defaultStartConfig!;
    }
    final cfg = StartConfig()
      ..storage = 'bolt'
      ..storageDir = Util.getStorageDir()
      ..refreshInterval = 0
      ..apiToken = '';
    if (!Util.supportUnixSocket()) {
      cfg
        ..network = 'tcp'
        ..address = '127.0.0.1:0';
    } else {
      cfg
        ..network = 'unix'
        ..address = '${(await getTemporaryDirectory()).path}/$unixSocketPath';
    }
    _defaultStartConfig = cfg;
    return cfg;
  }

  Future<DownloaderConfig> _loadDownloaderConfig() async {
    DownloaderConfig config;
    try {
      config = await ref.read(gopeedServiceProvider).getConfig();
    } catch (error, stackTrace) {
      logger.w('load downloader config failed', error, stackTrace);
      config = DownloaderConfig();
    }
    await _initDownloaderConfig(config);
    return config;
  }

  Future<void> _initDownloaderConfig(DownloaderConfig config) async {
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
      config.downloadDir = await _defaultDownloadDir();
    }
    if (extra.downloadCategories.isEmpty) {
      extra.downloadCategories = [
        DownloadCategory(
          name: '',
          path: path.join(config.downloadDir, 'Music'),
          isBuiltIn: true,
          nameKey: 'categoryMusic',
        ),
        DownloadCategory(
          name: '',
          path: path.join(config.downloadDir, 'Video'),
          isBuiltIn: true,
          nameKey: 'categoryVideo',
        ),
        DownloadCategory(
          name: '',
          path: path.join(config.downloadDir, 'Document'),
          isBuiltIn: true,
          nameKey: 'categoryDocument',
        ),
        DownloadCategory(
          name: '',
          path: path.join(config.downloadDir, 'Program'),
          isBuiltIn: true,
          nameKey: 'categoryProgram',
        ),
      ];
    }
    if (extra.githubMirror.mirrors.isEmpty) {
      extra.githubMirror.mirrors = [
        GithubMirror(type: GithubMirrorType.jsdelivr, url: 'https://fastly.jsdelivr.net/gh', isBuiltIn: true),
        GithubMirror(type: GithubMirrorType.ghProxy, url: 'https://fastgit.cc', isBuiltIn: true),
      ];
    }
  }

  Future<void> _initTrackerUpdate(DownloaderConfig config) async {
    final btExtra = config.extra.bt;
    final lastUpdateTime = btExtra.lastTrackerUpdateTime;
    if (!btExtra.autoUpdateTrackers ||
        (lastUpdateTime != null && !lastUpdateTime.isBefore(DateTime.now().subtract(const Duration(days: 1))))) {
      return;
    }
    try {
      await updateTrackers(config);
    } catch (error, stackTrace) {
      logger.w('tracker update failed', error, stackTrace);
    }
  }

  Future<void> updateTrackers(DownloaderConfig config) async {
    final btExtra = config.extra.bt;
    final trackers = <String>[];
    for (final url in btExtra.trackerSubscribeUrls) {
      final resp = await api.proxyRequest(url);
      final data = resp.data;
      if (resp.statusCode == 200 && data != null && data.isNotEmpty) {
        trackers.addAll(const LineSplitter().convert(data).where((line) => line.trim().isNotEmpty));
      }
    }
    btExtra.subscribeTrackers = trackers;
    btExtra.lastTrackerUpdateTime = DateTime.now();
    config.protocolConfig.bt.trackers = {...btExtra.subscribeTrackers, ...btExtra.customTrackers}.toList();
    await ref.read(gopeedServiceProvider).putConfig(config);

    final current = state.value;
    if (current != null) {
      state = AsyncValue.data(
        AppRuntimeState(
          startConfig: current.startConfig,
          runningPort: current.runningPort,
          downloaderConfig: config,
          localBackendStarted: current.localBackendStarted,
          startupError: current.startupError,
        ),
      );
    }
  }

  Future<String> _defaultDownloadDir() async {
    if (kIsWeb) {
      return './';
    }
    if (Util.isDesktop()) {
      return (await getDownloadsDirectory())?.path ?? './';
    }
    if (Util.isAndroid()) {
      return (await getExternalStorageDirectory())?.path ?? (await getApplicationDocumentsDirectory()).path;
    }
    if (Util.isIOS()) {
      return (await getApplicationDocumentsDirectory()).path;
    }
    return './';
  }

  String _runningAddress(StartConfig config, int runningPort) {
    if (config.network == 'unix') {
      return config.address;
    }
    return '${config.address.split(':').first}:$runningPort';
  }

  Future<void> saveStartConfig(StartConfig config) async {
    Database.instance.saveStartConfig(
      StartConfigEntity(network: config.network, address: config.address, apiToken: config.apiToken),
    );
  }
}

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
