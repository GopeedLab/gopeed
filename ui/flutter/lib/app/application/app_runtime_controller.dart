import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';

import '../../api/api.dart' as api;
import '../../api/model/downloader_config.dart';
import '../../core/capabilities/app_capabilities.dart';
import '../../core/common/api_server_state.dart';
import '../../core/common/start_config.dart';
import '../../core/common/task_event.dart';
import '../../core/entry/app_initializer.dart';
import '../../core/libgopeed_boot.dart';
import '../../core/network/gopeed/gopeed_transport.dart';
import '../../features/auth/application/web_auth_controller.dart';
import '../../l10n/l10n.dart';
import '../../util/log_util.dart';
import '../../util/analytics.dart';
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
    required this.apiServerState,
    required this.downloaderConfig,
    this.localBackendStarted = false,
    this.startupError,
  });

  final StartConfig startConfig;
  final ApiServerState apiServerState;
  final DownloaderConfig downloaderConfig;
  final bool localBackendStarted;
  final Object? startupError;

  int get runningPort => apiServerState.runningPort;

  String runningAddress() => apiServerState.runningAddress();
}

class AppRuntimeController extends AsyncNotifier<AppRuntimeState> {
  static StartConfig? _defaultStartConfig;
  Future<void> _preferenceWrites = Future<void>.value();

  @override
  Future<AppRuntimeState> build() async {
    ref.onDispose(() => unawaited(WebViewRpcService.instance.stop()));
    return _init();
  }

  Future<void> reloadConfig() async {
    final current = state.value;
    if (current == null) return;
    final config = await ref.read(gopeedServiceProvider).getConfig();
    final apiState = kIsWeb ? current.apiServerState : (await LibgopeedBoot.instance.getApiServerState()).state;
    final startConfig = _copyStartConfig(current.startConfig);
    if (kIsWeb) {
      // The Web client connection may point at a separate development server
      // from the page itself. Downloader config describes the remote process
      // and must not replace that client-side endpoint after reload.
      startConfig
        ..apiEnable = true
        ..mcpEnable = config.api.mcpEnable;
    } else {
      startConfig
        ..apiEnable = config.api.enable
        ..mcpEnable = config.api.mcpEnable
        ..network = config.api.network
        ..address = config.api.address
        ..apiToken = config.api.token;
    }
    state = AsyncValue.data(
      AppRuntimeState(
        startConfig: startConfig,
        apiServerState: apiState,
        downloaderConfig: config,
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
        apiServerState: current.apiServerState,
        downloaderConfig: DownloaderConfig.fromJson(config.toJson()),
        localBackendStarted: current.localBackendStarted,
        startupError: current.startupError,
      ),
    );
    unawaited(LocationKeepAliveCoordinator.instance.reconcile(enabled: config.extra.backgroundLocationKeepAlive));
  }

  Future<AppRuntimeState> _init() async {
    await AppInitializer.ensureStorageInitialized();
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
    var apiServerState = const ApiServerState(
      enabled: false,
      mcpEnabled: false,
      running: false,
      network: '',
      address: '',
      runningPort: 0,
      pendingApply: false,
      lastError: '',
    );
    var localBackendStarted = false;
    try {
      // Go's rest.Start is already idempotent: when its server is running it
      // returns the existing port before rebuilding or touching the Unix
      // socket. This is also what keeps Riverpod hot reload safe.
      runningPort = await LibgopeedBoot.instance.start(startConfig);
      await LibgopeedBoot.instance.subscribeTaskEvents({TaskEventType.done, TaskEventType.error});
      if (!kIsWeb) {
        final statusResult = await LibgopeedBoot.instance.getApiServerState();
        apiServerState = statusResult.state;
        if (statusResult.error.isNotEmpty) {
          logger.w('read API server state failed: ${statusResult.error}');
        }
      }
      localBackendStarted = true;
    } catch (error, stackTrace) {
      logger.e('libgopeed init failed', error, stackTrace);
      rethrow;
    }

    if (kIsWeb) {
      final webAuth = ref.read(webAuthControllerProvider);
      api.init('tcp', startConfig.address, '', onUnauthorized: webAuth.requireLogin);
    } else {
      api.init(startConfig.network, _runningAddress(startConfig, runningPort), startConfig.apiToken);
    }

    final config = await _loadDownloaderConfig();
    if (!kIsWeb) {
      startConfig
        ..apiEnable = config.api.enable
        ..mcpEnable = config.api.mcpEnable
        ..network = config.api.network
        ..address = config.api.address
        ..apiToken = config.api.token;
    }
    unawaited(_initTrackerUpdate(config));
    unawaited(_initAndroidForegroundService(appLocalizationsFor(config.extra.locale)));
    await _initAnalytics(config);

    final result = AppRuntimeState(
      startConfig: startConfig,
      apiServerState: apiServerState,
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
    final cfg = StartConfig()
      ..network = defaultCfg.network
      ..address = defaultCfg.address
      ..apiToken = defaultCfg.apiToken
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
    if (kIsWeb) {
      cfg
        ..network = 'tcp'
        ..address = webApiBaseUrl(Uri.base);
      _defaultStartConfig = cfg;
      return cfg;
    }
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

  Future<void> _initAnalytics(DownloaderConfig config) async {
    if (!AnalyticsConfig.isConfigured || !config.extra.analyticsEnabled) return;
    try {
      await Analytics.instance.init(fallbackClientId: config.extra.analyticsClientId);
      if (config.extra.analyticsClientId.isEmpty && Analytics.instance.clientId.isNotEmpty) {
        config.extra.analyticsClientId = Analytics.instance.clientId;
        await ref.read(gopeedServiceProvider).putConfig(config);
      }
      unawaited(Analytics.instance.logAppOpen());
    } catch (error, stackTrace) {
      logger.w('GA4 init failed', error, stackTrace);
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
          apiServerState: current.apiServerState,
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

  Future<void> saveApiServerConfig(StartConfig config) async {
    if (kIsWeb) return;
    await _persistApiServerConfig(config);
  }

  Future<void> startApiServer() async {
    if (kIsWeb) return;
    await _persistApiServerEnabled(true);
    await _applyApiServerOperation(LibgopeedBoot.instance.startApiServer());
  }

  Future<void> stopApiServer() async {
    if (kIsWeb) return;
    await _persistApiServerEnabled(false);
    await _applyApiServerOperation(LibgopeedBoot.instance.stopApiServer());
  }

  Future<void> restartApiServer(StartConfig config) async {
    if (kIsWeb) return;
    final desired = _copyStartConfig(config)..apiEnable = true;
    await _persistApiServerConfig(desired);
    await _applyApiServerOperation(LibgopeedBoot.instance.restartApiServer());
  }

  Future<void> refreshApiServerState() async {
    if (kIsWeb) return;
    await _applyApiServerOperation(LibgopeedBoot.instance.getApiServerState());
  }

  Future<void> _persistApiServerConfig(StartConfig config) async {
    final current = state.value;
    if (current == null) return;
    final nextConfig = _copyStartConfig(config);
    final downloaderConfig = await ref.read(gopeedServiceProvider).getConfig();
    downloaderConfig.api
      ..enable = nextConfig.apiEnable
      ..mcpEnable = nextConfig.mcpEnable
      ..network = nextConfig.network
      ..address = nextConfig.address
      ..token = nextConfig.apiToken;
    await ref.read(gopeedServiceProvider).putConfig(downloaderConfig);
    final statusResult = await LibgopeedBoot.instance.getApiServerState();
    _replaceApiRuntime(nextConfig, downloaderConfig, statusResult.state);
    if (statusResult.error.isNotEmpty) {
      throw StateError(statusResult.error);
    }
  }

  Future<void> _persistApiServerEnabled(bool enabled) async {
    final current = state.value;
    if (current == null) return;
    final downloaderConfig = await ref.read(gopeedServiceProvider).getConfig();
    downloaderConfig.api.enable = enabled;
    await ref.read(gopeedServiceProvider).putConfig(downloaderConfig);
    final nextConfig = _copyStartConfig(current.startConfig)..apiEnable = enabled;
    final statusResult = await LibgopeedBoot.instance.getApiServerState();
    _replaceApiRuntime(nextConfig, downloaderConfig, statusResult.state);
    if (statusResult.error.isNotEmpty) {
      throw StateError(statusResult.error);
    }
  }

  Future<void> _applyApiServerOperation(Future<ApiServerOperationResult> operation) async {
    final result = await operation;
    final current = state.value;
    if (current != null) {
      _replaceApiRuntime(current.startConfig, current.downloaderConfig, result.state);
    }
    if (result.error.isNotEmpty) {
      throw StateError(result.error);
    }
  }

  void _replaceApiRuntime(StartConfig config, DownloaderConfig downloaderConfig, ApiServerState apiState) {
    final current = state.value;
    if (current == null) return;
    state = AsyncValue.data(
      AppRuntimeState(
        startConfig: _copyStartConfig(config),
        apiServerState: apiState,
        downloaderConfig: DownloaderConfig.fromJson(downloaderConfig.toJson()),
        localBackendStarted: current.localBackendStarted,
        startupError: current.startupError,
      ),
    );
  }

  StartConfig _copyStartConfig(StartConfig config) {
    return StartConfig()
      ..network = config.network
      ..address = config.address
      ..apiEnable = config.apiEnable
      ..mcpEnable = config.mcpEnable
      ..storage = config.storage
      ..storageDir = config.storageDir
      ..refreshInterval = config.refreshInterval
      ..apiToken = config.apiToken
      ..webViewRpcConfig = config.webViewRpcConfig;
  }

  Future<void> updateExtraConfig(void Function(ExtraConfig extra) mutation) {
    final operation = _preferenceWrites.then((_) async {
      final current = state.value;
      if (current == null) return;
      final config = await ref.read(gopeedServiceProvider).getConfig();
      mutation(config.extra);
      await ref.read(gopeedServiceProvider).putConfig(config);
      replaceDownloaderConfig(config);
    });
    _preferenceWrites = operation.then<void>((_) {}, onError: (_, _) {});
    return operation;
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
