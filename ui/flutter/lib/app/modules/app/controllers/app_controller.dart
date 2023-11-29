import 'dart:async';
import 'dart:convert';
import 'dart:ui';

import 'package:app_links/app_links.dart';
import 'package:flutter/material.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:get/get.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:uri_to_file/uri_to_file.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:window_manager/window_manager.dart';

import '../../../../api/api.dart';
import '../../../../api/model/downloader_config.dart';
import '../../../../core/common/start_config.dart';
import '../../../../generated/locales.g.dart';
import '../../../../util/locale_manager.dart';
import '../../../../util/log_util.dart';
import '../../../../util/package_info.dart';
import '../../../../util/util.dart';
import '../../../routes/app_pages.dart';

const _startConfigNetwork = "start.network";
const _startConfigAddress = "start.address";
const _startConfigApiToken = "start.apiToken";

const unixSocketPath = 'gopeed.sock';

const allTrackerSubscribeUrls = [
  'https://github.com/ngosang/trackerslist/raw/master/trackers_all.txt',
  'https://github.com/ngosang/trackerslist/raw/master/trackers_all_http.txt',
  'https://github.com/ngosang/trackerslist/raw/master/trackers_all_https.txt',
  'https://github.com/ngosang/trackerslist/raw/master/trackers_all_ip.txt',
  'https://github.com/ngosang/trackerslist/raw/master/trackers_all_udp.txt',
  'https://github.com/ngosang/trackerslist/raw/master/trackers_all_ws.txt',
  'https://github.com/ngosang/trackerslist/raw/master/trackers_best.txt',
  'https://github.com/ngosang/trackerslist/raw/master/trackers_best_ip.txt',
  'https://github.com/XIU2/TrackersListCollection/raw/master/all.txt',
  'https://github.com/XIU2/TrackersListCollection/raw/master/best.txt',
  'https://github.com/XIU2/TrackersListCollection/raw/master/http.txt',
];
const allTrackerCdns = [
  // jsdelivr: https://fastly.jsdelivr.net/gh/ngosang/trackerslist/trackers_all.txt
  ["https://fastly.jsdelivr.net/gh", r".*github.com(/.*)/raw/master(/.*)"],
  // ghproxy: https://ghproxy.com/https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt
  [
    "https://ghproxy.com/https://raw.githubusercontent.com",
    r".*github.com(/.*)/raw(/.*)"
  ]
];
final allTrackerSubscribeUrlCdns = Map.fromIterable(allTrackerSubscribeUrls,
    key: (v) => v as String,
    value: (v) {
      final ret = [v as String];
      for (final cdn in allTrackerCdns) {
        final reg = RegExp(cdn[1]);
        final match = reg.firstMatch(v.toString());
        var matchStr = "";
        for (var i = 1; i <= match!.groupCount; i++) {
          matchStr += match.group(i)!;
        }
        ret.add("${cdn[0]}$matchStr");
      }
      return ret;
    });

class AppController extends GetxController with WindowListener, TrayListener {
  static StartConfig? _defaultStartConfig;

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

    _initForegroundTask().onError((error, stackTrace) =>
        logger.w("initForegroundTask error", error, stackTrace));
  }

  @override
  void onClose() {
    _linkSubscription?.cancel();
    trayManager.removeListener(this);
  }

  @override
  void onWindowClose() async {
    final isPreventClose = await windowManager.isPreventClose();
    if (isPreventClose) {
      windowManager.hide();
    }
  }

  @override
  void onTrayIconMouseDown() {
    windowManager.show();
  }

  @override
  void onTrayIconRightMouseDown() {
    trayManager.popUpContextMenu();
  }

  Future<void> _initDeepLinks() async {
    // currently only support android
    if (!Util.isAndroid()) {
      return;
    }

    _appLinks = AppLinks();

    // Handle link when app is in warm state (front or background)
    _linkSubscription = _appLinks.uriLinkStream.listen((uri) async {
      await _toCreate(uri);
    });

    // Check initial link if app was in cold state (terminated)
    final uri = await _appLinks.getInitialAppLink();
    if (uri != null) {
      await _toCreate(uri);
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
    } else {
      await trayManager.setIcon('assets/tray_icon/icon.png');
    }
    final menu = Menu(items: [
      MenuItem(
        label: "create".tr,
        onClick: (menuItem) async => {
          await windowManager.show(),
          await Get.rootDelegate.offAndToNamed(Routes.CREATE),
        },
      ),
      MenuItem.separator(),
      MenuItem(
        label: "startAll".tr,
        onClick: (menuItem) async => {continueAllTasks()},
      ),
      MenuItem(
        label: "pauseAll".tr,
        onClick: (menuItem) async => {pauseAllTasks()},
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
          launchUrl(
              Uri.parse(
                  "https://github.com/GopeedLab/gopeed/blob/main/.donate/index.md#donate"),
              mode: LaunchMode.externalApplication)
        },
      ),
      MenuItem(
        label: '${"version".tr}（${packageInfo.version}）',
      ),
      MenuItem.separator(),
      MenuItem(
        label: 'exit'.tr,
        onClick: (menuItem) => {windowManager.destroy()},
      ),
    ]);
    if (!Util.isLinux()) {
      // Linux seems not support setToolTip, refer to: https://github.com/GopeedLab/gopeed/issues/241
      await trayManager.setToolTip('Gopeed');
    }
    await trayManager.setContextMenu(menu);
    trayManager.addListener(this);
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
          iconData: const NotificationIconData(
            resType: ResourceType.mipmap,
            resPrefix: ResourcePrefix.ic,
            name: 'launcher',
          )),
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
      );
    }
  }

  Future<void> _toCreate(Uri uri) async {
    final path = uri.scheme == "magnet"
        ? uri.toString()
        : (await toFile(uri.toString())).path;
    await Get.rootDelegate.offAndToNamed(Routes.CREATE, arguments: path);
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

  Future<void> loadStartConfig() async {
    final defaultCfg = await _initDefaultStartConfig();
    final prefs = await SharedPreferences.getInstance();
    startConfig.value.network =
        prefs.getString(_startConfigNetwork) ?? defaultCfg.network;
    startConfig.value.address =
        prefs.getString(_startConfigAddress) ?? defaultCfg.address;
    startConfig.value.apiToken =
        prefs.getString(_startConfigApiToken) ?? defaultCfg.apiToken;
  }

  Future<void> loadDownloaderConfig() async {
    try {
      downloaderConfig.value = await getConfig();
    } catch (e) {
      logger.w("load downloader config fail", e);
      downloaderConfig.value = DownloaderConfig();
    }
    await _initDownloaderConfig();
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

  Future<void> trackerUpdateOnStart() async {
    final btExtConfig = downloaderConfig.value.extra.bt;
    final lastUpdateTime = btExtConfig.lastTrackerUpdateTime;
    // if last update time is null or more than 1 day, update trackers
    if (lastUpdateTime == null ||
        lastUpdateTime.difference(DateTime.now()).inDays < 0) {
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
    if (config.protocolConfig.http.connections == 0) {
      config.protocolConfig.http.connections = 16;
    }
    final extra = config.extra;
    if (extra.themeMode.isEmpty) {
      extra.themeMode = ThemeMode.dark.name;
    }
    if (extra.locale.isEmpty) {
      final systemLocale = getLocaleKey(PlatformDispatcher.instance.locale);
      extra.locale = AppTranslation.translations.containsKey(systemLocale)
          ? systemLocale
          : getLocaleKey(fallbackLocale);
    }
    if (extra.bt.trackerSubscribeUrls.isEmpty) {
      // default select all tracker subscribe urls
      extra.bt.trackerSubscribeUrls.addAll(allTrackerSubscribeUrls);
    }

    if (config.downloadDir.isEmpty) {
      if (Util.isDesktop()) {
        config.downloadDir = (await getDownloadsDirectory())?.path ?? "./";
      } else if (Util.isAndroid()) {
        config.downloadDir = (await getExternalStorageDirectory())?.path ??
            (await getApplicationDocumentsDirectory()).path;
        return;
      } else if (Util.isIOS()) {
        config.downloadDir = (await getApplicationDocumentsDirectory()).path;
      } else {
        config.downloadDir = './';
      }
    }
  }

  Future<void> saveConfig() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_startConfigNetwork, startConfig.value.network);
    await prefs.setString(_startConfigAddress, startConfig.value.address);
    await prefs.setString(_startConfigApiToken, startConfig.value.apiToken);
    await putConfig(downloaderConfig.value);
  }
}
