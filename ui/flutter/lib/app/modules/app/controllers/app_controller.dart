import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:ui';

import 'package:app_links/app_links.dart';
import 'package:flutter/material.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:get/get.dart';
import 'package:launch_at_startup/launch_at_startup.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
import 'package:share_handler/share_handler.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:uri_to_file/uri_to_file.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:window_manager/window_manager.dart';

import '../../../../api/api.dart';
import '../../../../api/model/create_task.dart';
import '../../../../api/model/downloader_config.dart';
import '../../../../api/model/request.dart';
import '../../../../core/common/start_config.dart';
import '../../../../core/libgopeed_boot.dart';
import '../../../../database/database.dart';
import '../../../../database/entity.dart';
import '../../../../i18n/message.dart';
import '../../../../main.dart';
import '../../../../util/github_mirror.dart';
import '../../../../util/locale_manager.dart';
import '../../../../util/log_util.dart';
import '../../../../util/package_info.dart';
import '../../../../util/updater.dart';
import '../../../../util/util.dart';
import '../../../routes/app_pages.dart';
import '../../../rpc/rpc.dart';
import '../../redirect/views/redirect_view.dart';

const unixSocketPath = 'gopeed.sock';

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
final allTrackerSubscribeUrlCdns = {
  for (var v in allTrackerSubscribeUrls)
    v: githubMirrorUrls(v, MirrorType.githubSource)
};

class AppController extends GetxController with WindowListener, TrayListener {
  static StartConfig? _defaultStartConfig;

  final autoStartup = false.obs;
  final startConfig = StartConfig().obs;
  final runningPort = 0.obs;
  final downloaderConfig = DownloaderConfig().obs;

  late AppLinks _appLinks;
  StreamSubscription<Uri>? _linkSubscription;

  @override
  void onReady() {
    super.onReady();

    _initDeepLinks().onError((error, stackTrace) =>
        logger.w("initDeepLinks error", error, stackTrace));

    _initWindows().onError((error, stackTrace) =>
        logger.w("initWindows error", error, stackTrace));

    _initTray().onError(
        (error, stackTrace) => logger.w("initTray error", error, stackTrace));

    _initRpcServer().onError((error, stackTrace) =>
        logger.w("initRpcServer error", error, stackTrace));

    _initForegroundTask().onError((error, stackTrace) =>
        logger.w("initForegroundTask error", error, stackTrace));

    _initTrackerUpdate().onError((error, stackTrace) =>
        logger.w("initTrackerUpdate error", error, stackTrace));

    _initLaunchAtStartup().onError((error, stackTrace) =>
        logger.w("initLaunchAtStartup error", error, stackTrace));

    _initCheckUpdate().onError((error, stackTrace) =>
        logger.w("initCheckUpdate error", error, stackTrace));
  }

  @override
  void onClose() {
    _linkSubscription?.cancel();
    trayManager.removeListener(this);
    LibgopeedBoot.instance.stop();
  }

  @override
  void onWindowClose() async {
    final isPreventClose = await windowManager.isPreventClose();
    if (isPreventClose) {
      windowManager.hide();
    }
  }

  // According to the system_manager document, make sure to call setState once on the onWindowFocus event.
  @override
  void onWindowFocus() {
    refresh();
    if (Util.isMacos() && Database.instance.getRunAsMenubarApp()) {
      windowManager.setSkipTaskbar(true);
    }
  }

  @override
  void onTrayIconMouseDown() {
    windowManager.show();
  }

  @override
  void onTrayIconRightMouseDown() {
    trayManager.popUpContextMenu(bringAppToFront: true);
  }

  @override
  void onWindowMaximize() {
    Database.instance.saveWindowState(WindowStateEntity(isMaximized: true));
  }

  @override
  void onWindowUnmaximize() {
    Database.instance.saveWindowState(WindowStateEntity(isMaximized: false));
  }

  final _windowsResizeSave = Util.debounce(() async {
    final size = await windowManager.getSize();
    Database.instance.saveWindowState(
        WindowStateEntity(width: size.width, height: size.height));
  }, 500);

  @override
  void onWindowResize() {
    _windowsResizeSave();
  }

  Future<void> _initDeepLinks() async {
    if (Util.isWeb()) {
      return;
    }

    // Handle deep link
    () async {
      _appLinks = AppLinks();

      // Handle link when app is in warm state (front or background)
      _linkSubscription = _appLinks.uriLinkStream.listen((uri) async {
        await _handleDeepLink(uri);
      });

      // Check initial link if app was in cold state (terminated)
      final uri = await _appLinks.getInitialLink();
      if (uri != null) {
        await _handleDeepLink(uri);
      }
    }();

    // Handle shared media, e.g. shared link from browser
    if (Util.isMobile()) {
      () async {
        final handler = ShareHandlerPlatform.instance;

        handler.sharedMediaStream.listen((SharedMedia media) {
          if (media.content?.isNotEmpty == true) {
            final uri = Uri.parse(media.content!);
            // content uri will be handled by the app_links plugin
            if (uri.scheme != "content") {
              _handleDeepLink(uri);
            }
          }
        });

        final media = await handler.getInitialSharedMedia();
        if (media?.content?.isNotEmpty == true) {
          _handleDeepLink(Uri.parse(media!.content!));
        }
      }();
    }
  }

  Future<void> _initWindows() async {
    if (!Util.isDesktop()) {
      return;
    }
    windowManager.addListener(this);
  }

  Future<void> _initTray() async {
    if (!Util.isDesktop()) {
      return;
    }
    if (Util.isWindows()) {
      await trayManager.setIcon('assets/tray_icon/icon.ico');
    } else if (Util.isMacos()) {
      await trayManager.setIcon('assets/tray_icon/icon_mac.png',
          isTemplate: true);
    } else if (Platform.environment.containsKey('FLATPAK_ID') ||
        Platform.environment.containsKey('SNAP')) {
      await trayManager.setIcon('com.gopeed.Gopeed');
    } else {
      await trayManager.setIcon('assets/tray_icon/icon.png');
    }
    final menu = Menu(items: [
      MenuItem(
        label: "show".tr,
        onClick: (menuItem) async => {
          await windowManager.show(),
        },
      ),
      MenuItem.separator(),
      MenuItem(
        label: "create".tr,
        onClick: (menuItem) async => {
          await windowManager.show(),
          await Get.rootDelegate.offAndToNamed(Routes.CREATE),
        },
      ),
      MenuItem(
        label: "startAll".tr,
        onClick: (menuItem) async => {continueAllTasks(null)},
      ),
      MenuItem(
        label: "pauseAll".tr,
        onClick: (menuItem) async => {pauseAllTasks(null)},
      ),
      MenuItem(
        label: 'setting'.tr,
        onClick: (menuItem) async => {
          await windowManager.show(),
          await Get.rootDelegate.offAndToNamed(Routes.SETTING),
        },
      ),
      MenuItem.separator(),
      MenuItem(
        label: 'donate'.tr,
        onClick: (menuItem) => {
          launchUrl(Uri.parse("https://docs.gopeed.com/donate.html"),
              mode: LaunchMode.externalApplication)
        },
      ),
      MenuItem(
        label: '${"version".tr}（${packageInfo.version}）',
      ),
      MenuItem.separator(),
      MenuItem(
        label: 'exit'.tr,
        onClick: (menuItem) async {
          try {
            await LibgopeedBoot.instance.stop();
          } catch (e) {
            logger.w("libgopeed stop fail", e);
          }
          windowManager.destroy();
        },
      ),
    ]);
    if (!Util.isLinux()) {
      // Linux seems not support setToolTip, refer to: https://github.com/GopeedLab/gopeed/issues/241
      await trayManager.setToolTip('Gopeed');
    }
    await trayManager.setContextMenu(menu);
    trayManager.addListener(this);
  }

  Future<void> _initRpcServer() async {
    if (!Util.isDesktop()) {
      return;
    }
    try {
      await startRpcServer({
        "/create": (ctx) async {
          final meta =
              ctx.request.headers["X-Gopeed-Host-Meta"]?.firstOrNull ?? "{}";
          final jsonMeta = jsonDecode(meta);
          final silent = jsonMeta['silent'] as bool? ?? false;
          final params = await ctx.readText();
          final createTaskParams = _decodeToCreatTaskParams(params);
          if (!silent) {
            await windowManager.show();
            _handleToCreate0(createTaskParams);
          } else {
            try {
              await createTask(createTaskParams);
            } catch (e) {
              logger.w(
                  "create task from extension fail", e, StackTrace.current);
            }
          }
        },
      });
    } catch (e) {
      logger.w("start rpc server fail", e, StackTrace.current);
    }
  }

  Future<void> _initForegroundTask() async {
    if (!Util.isMobile()) {
      return;
    }

    FlutterForegroundTask.init(
      androidNotificationOptions: AndroidNotificationOptions(
        channelId: 'gopeed_service',
        channelName: 'Gopeed Background Service',
        channelImportance: NotificationChannelImportance.LOW,
        showWhen: true,
        priority: NotificationPriority.LOW,
      ),
      iosNotificationOptions: const IOSNotificationOptions(
        showNotification: true,
        playSound: false,
      ),
      foregroundTaskOptions: const ForegroundTaskOptions(
        interval: 5000,
        isOnceEvent: false,
        autoRunOnBoot: true,
        allowWakeLock: true,
        allowWifiLock: true,
      ),
    );

    if (await FlutterForegroundTask.isRunningService) {
      FlutterForegroundTask.restartService();
    } else {
      FlutterForegroundTask.startService(
        notificationTitle: "serviceTitle".tr,
        notificationText: "serviceText".tr,
        notificationIcon: const NotificationIconData(
          resType: ResourceType.mipmap,
          resPrefix: ResourcePrefix.ic,
          name: 'launcher',
        ),
      );
    }
  }

  Future<void> _handleDeepLink(Uri uri) async {
    if (uri.scheme == "gopeed") {
      if (uri.path == "/create") {
        final params = uri.queryParameters["params"];
        if (params?.isNotEmpty == true) {
          _handleToCreate(params!);
          return;
        }
        Get.rootDelegate.offAndToNamed(Routes.CREATE);
        return;
      }
      Get.rootDelegate.offAndToNamed(Routes.HOME);
      return;
    }

    String path;
    if (uri.scheme == "magnet" ||
        uri.scheme == "http" ||
        uri.scheme == "https") {
      path = uri.toString();
    } else if (uri.scheme == "file") {
      path =
          Util.isWindows() ? Uri.decodeFull(uri.path.substring(1)) : uri.path;
    } else {
      path = (await toFile(uri.toString())).path;
    }
    Get.rootDelegate.offAndToNamed(Routes.REDIRECT,
        arguments: RedirectArgs(Routes.CREATE,
            arguments: CreateTask(req: Request(url: path))));
  }

  String runningAddress() {
    if (startConfig.value.network == 'unix') {
      return startConfig.value.address;
    }
    return '${startConfig.value.address.split(':').first}:$runningPort';
  }

  Future<StartConfig> _initDefaultStartConfig() async {
    if (_defaultStartConfig != null) {
      return _defaultStartConfig!;
    }
    _defaultStartConfig = StartConfig();
    if (!Util.supportUnixSocket()) {
      // not support unix socket, use tcp
      _defaultStartConfig!.network = "tcp";
      _defaultStartConfig!.address = "127.0.0.1:0";
    } else {
      _defaultStartConfig!.network = "unix";
      _defaultStartConfig!.address =
          "${(await getTemporaryDirectory()).path}/$unixSocketPath";
    }
    _defaultStartConfig!.apiToken = '';
    return _defaultStartConfig!;
  }

  Future<StartConfig> loadStartConfig() async {
    final defaultCfg = await _initDefaultStartConfig();
    final saveCfg = Database.instance.getStartConfig();
    startConfig.value.network = saveCfg?.network ?? defaultCfg.network;
    startConfig.value.address = saveCfg?.address ?? defaultCfg.address;
    startConfig.value.apiToken = saveCfg?.apiToken ?? defaultCfg.apiToken;
    return startConfig.value;
  }

  Future<DownloaderConfig> loadDownloaderConfig() async {
    try {
      downloaderConfig.value = await getConfig();
    } catch (e) {
      logger.w("load downloader config fail", e, StackTrace.current);
      downloaderConfig.value = DownloaderConfig();
    }
    await _initDownloaderConfig();
    return downloaderConfig.value;
  }

  Future<void> trackerUpdate() async {
    final btExtConfig = downloaderConfig.value.extra.bt;
    final result = <String>[];
    for (var u in btExtConfig.trackerSubscribeUrls) {
      final cdns = allTrackerSubscribeUrlCdns[u];
      if (cdns == null) {
        continue;
      }
      try {
        final trackers =
            await Util.anyOk(cdns.map((cdn) => _fetchTrackers(cdn)));
        result.addAll(trackers);
      } catch (e) {
        logger.w("subscribe trackers fail, url: $u", e);
        return;
      }
    }
    btExtConfig.subscribeTrackers.clear();
    btExtConfig.subscribeTrackers.addAll(result);
    downloaderConfig.update((val) {
      val!.extra.bt.lastTrackerUpdateTime = DateTime.now();
    });
    refreshTrackers();

    await saveConfig();
  }

  refreshTrackers() {
    final btConfig = downloaderConfig.value.protocolConfig.bt;
    final btExtConfig = downloaderConfig.value.extra.bt;
    btConfig.trackers.clear();
    btConfig.trackers.addAll(btExtConfig.subscribeTrackers);
    btConfig.trackers.addAll(btExtConfig.customTrackers);
    // remove duplicate
    btConfig.trackers.toSet().toList();
  }

  Future<void> _initTrackerUpdate() async {
    final btExtConfig = downloaderConfig.value.extra.bt;
    final lastUpdateTime = btExtConfig.lastTrackerUpdateTime;
    // if last update time is null or more than 1 day, update trackers
    if (btExtConfig.autoUpdateTrackers &&
        (lastUpdateTime == null ||
            lastUpdateTime.difference(DateTime.now()).inDays < 0)) {
      try {
        await trackerUpdate();
      } catch (e) {
        logger.w("tracker update fail", e);
      }
    }
  }

  Future<List<String>> _fetchTrackers(String subscribeUrl) async {
    final resp = await proxyRequest(subscribeUrl);
    if (resp.statusCode != 200) {
      throw Exception(
          'Failed to get trackers, status code: ${resp.statusCode}');
    }
    if (resp.data == null || resp.data!.isEmpty) {
      throw Exception('Failed to get trackers, data is null');
    }
    const ls = LineSplitter();
    return ls.convert(resp.data!).where((e) => e.isNotEmpty).toList();
  }

  _initDownloaderConfig() async {
    final config = downloaderConfig.value;
    final extra = config.extra;
    if (extra.themeMode.isEmpty) {
      extra.themeMode = ThemeMode.dark.name;
    }
    if (extra.locale.isEmpty) {
      final systemLocale = getLocaleKey(PlatformDispatcher.instance.locale);
      extra.locale = messages.keys.containsKey(systemLocale)
          ? systemLocale
          : getLocaleKey(fallbackLocale);
    }
    if (extra.bt.trackerSubscribeUrls.isEmpty) {
      // default select all tracker subscribe urls
      extra.bt.trackerSubscribeUrls.addAll(allTrackerSubscribeUrls);
    }

    final proxy = config.proxy;
    if (proxy.scheme.isEmpty) {
      proxy.scheme = 'http';
    }

    if (config.downloadDir.isEmpty) {
      if (Util.isDesktop()) {
        config.downloadDir = (await getDownloadsDirectory())?.path ?? "./";
      } else if (Util.isAndroid()) {
        config.downloadDir = (await getExternalStorageDirectory())?.path ??
            (await getApplicationDocumentsDirectory()).path;
      } else if (Util.isIOS()) {
        config.downloadDir = (await getApplicationDocumentsDirectory()).path;
      } else {
        config.downloadDir = './';
      }
    }

    // Initialize default download categories if empty
    if (extra.downloadCategories.isEmpty) {
      _initDefaultDownloadCategories();
    }

    // Initialize default GitHub mirrors if empty
    if (extra.githubMirror.mirrors.isEmpty) {
      _initDefaultGithubMirrors();
    }
  }

  void _initDefaultDownloadCategories() {
    final extra = downloaderConfig.value.extra;
    final downloadDir = downloaderConfig.value.downloadDir;

    // Add default built-in categories with i18n keys
    // No need to set initial name value, it will be retrieved via nameKey
    extra.downloadCategories = [
      DownloadCategory(
        name: '',
        path: path.join(downloadDir, 'Music'),
        isBuiltIn: true,
        nameKey: 'categoryMusic',
      ),
      DownloadCategory(
        name: '',
        path: path.join(downloadDir, 'Video'),
        isBuiltIn: true,
        nameKey: 'categoryVideo',
      ),
      DownloadCategory(
        name: '',
        path: path.join(downloadDir, 'Document'),
        isBuiltIn: true,
        nameKey: 'categoryDocument',
      ),
      DownloadCategory(
        name: '',
        path: path.join(downloadDir, 'Program'),
        isBuiltIn: true,
        nameKey: 'categoryProgram',
      ),
    ];
  }

  void _initDefaultGithubMirrors() {
    final extra = downloaderConfig.value.extra;

    // Add default built-in GitHub mirrors
    extra.githubMirror.mirrors = [
      GithubMirror(
        type: GithubMirrorType.jsdelivr,
        url: 'https://fastly.jsdelivr.net/gh',
        isBuiltIn: true,
      ),
      GithubMirror(
        type: GithubMirrorType.ghProxy,
        url: 'https://fastgit.cc',
        isBuiltIn: true,
      ),
    ];
  }

  Future<void> _initLaunchAtStartup() async {
    if (!Util.isWindows() && !Util.isLinux()) {
      return;
    }
    launchAtStartup.setup(
        appName: packageInfo.appName,
        appPath: Platform.resolvedExecutable,
        args: ['--${Args.flagHidden}']);
    autoStartup.value = await launchAtStartup.isEnabled();
  }

  Future<void> _initCheckUpdate() async {
    // Check if auto check update is enabled
    if (!downloaderConfig.value.extra.notifyWhenNewVersion) {
      return;
    }

    final versionInfo = await checkUpdate();
    if (versionInfo != null) {
      await showUpdateDialog(Get.context!, versionInfo);
    }
  }

  Future<void> saveConfig() async {
    Database.instance.saveStartConfig(StartConfigEntity(
        network: startConfig.value.network,
        address: startConfig.value.address,
        apiToken: startConfig.value.apiToken));
    await putConfig(downloaderConfig.value);
  }

  CreateTask _decodeToCreatTaskParams(String params) {
    final safeParams = params.replaceAll('"', "").replaceAll(" ", "+");
    final paramsJson =
        String.fromCharCodes(base64Decode(base64.normalize(safeParams)));
    return CreateTask.fromJson(jsonDecode(paramsJson));
  }

  _handleToCreate(String params) {
    final createTaskParams = _decodeToCreatTaskParams(params);
    _handleToCreate0(createTaskParams);
  }

  _handleToCreate0(CreateTask createTaskParams) {
    Get.rootDelegate.offAndToNamed(Routes.REDIRECT,
        arguments: RedirectArgs(Routes.CREATE, arguments: createTaskParams));
  }
}
