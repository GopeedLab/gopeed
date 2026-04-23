import 'package:args/args.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:get/get.dart';
import 'package:gopeed/util/analytics.dart';
import 'package:hotkey_manager/hotkey_manager.dart';
import 'package:window_manager/window_manager.dart';

import 'api/api.dart' as api;
import 'app/services/browser_download_popup.dart';
import 'app/modules/app/controllers/app_controller.dart';
import 'app/modules/app/views/app_view.dart';
import 'core/libgopeed_boot.dart';
import 'database/database.dart';
import 'i18n/message.dart';
import 'util/browser_extension_host/browser_extension_host.dart';
import 'util/locale_manager.dart';
import 'util/log_util.dart';
import 'util/package_info.dart';
import 'util/scheme_register/scheme_register.dart';
import 'util/updater.dart';
import 'util/util.dart';

class StartupArgs {
  static const flagHidden = "hidden";
  static const flagDownloadPopup = "download-popup";
  static const popupTaskId = "popup-task-id";
  static const popupNetwork = "popup-network";
  static const popupAddress = "popup-address";
  static const popupApiToken = "popup-api-token";
  static const popupThemeMode = "popup-theme-mode";
  static const popupLocale = "popup-locale";

  /// Command line --hidden flag (for auto-start)
  bool hiddenFromArgs = false;
  bool isDownloadPopup = false;
  String popupTaskIdValue = '';
  String popupNetworkValue = '';
  String popupAddressValue = '';
  String popupApiTokenValue = '';
  String popupThemeModeValue = ThemeMode.system.name;
  String popupLocaleValue =
      fallbackLocale.languageCode + '_' + (fallbackLocale.countryCode ?? 'US');

  StartupArgs._();

  /// Parse from command line arguments only
  static StartupArgs parse(List<String> arguments) {
    final args = StartupArgs._();
    try {
      final parser = ArgParser()
        ..addFlag(flagHidden)
        ..addFlag(flagDownloadPopup)
        ..addOption(popupTaskId)
        ..addOption(popupNetwork)
        ..addOption(popupAddress)
        ..addOption(popupApiToken)
        ..addOption(popupThemeMode)
        ..addOption(popupLocale);
      final results = parser.parse(arguments);
      args.hiddenFromArgs = results.flag(flagHidden);
      args.isDownloadPopup = results.flag(flagDownloadPopup);
      args.popupTaskIdValue = results.option(popupTaskId) ?? '';
      args.popupNetworkValue = results.option(popupNetwork) ?? '';
      args.popupAddressValue = results.option(popupAddress) ?? '';
      args.popupApiTokenValue = results.option(popupApiToken) ?? '';
      args.popupThemeModeValue =
          results.option(popupThemeMode) ?? ThemeMode.system.name;
      args.popupLocaleValue =
          results.option(popupLocale) ?? getLocaleKey(fallbackLocale);
    } catch (e) {
      // ignore parse errors
    }
    return args;
  }
}

void main(List<String> arguments) async {
  WidgetsFlutterBinding.ensureInitialized();

  final args = StartupArgs.parse(arguments);

  await init(args);
  if (args.isDownloadPopup) {
    await appendPopupDebugLog(
        'main enter popup mode task=${args.popupTaskIdValue} network=${args.popupNetworkValue} address=${args.popupAddressValue}');
    runApp(BrowserDownloadPopupApp(
      taskId: args.popupTaskIdValue,
      themeMode: args.popupThemeModeValue,
      localeKey: args.popupLocaleValue,
    ));
    return;
  }
  onStart();

  runApp(const AppView());
}

Future<void> init(StartupArgs args) async {
  // Note: WidgetsFlutterBinding.ensureInitialized() is already called in main()
  if (Util.isMobile()) {
    FlutterForegroundTask.initCommunicationPort();
  }
  await Util.initStorageDir();
  if (Util.isDesktop()) {
    await windowManager.ensureInitialized();
    if (!args.isDownloadPopup) {
      await Database.instance.init();
      final windowState = Database.instance.getWindowState();

      // Check if menubar mode is enabled (only for macOS)
      final runAsMenubarApp =
          Util.isMacos() && Database.instance.getRunAsMenubarApp();

      final windowOptions = WindowOptions(
        size: Size(windowState?.width ?? 800, windowState?.height ?? 600),
        center: true,
        skipTaskbar: runAsMenubarApp,
      );
      await windowManager.waitUntilReadyToShow(windowOptions, () async {
        await windowManager.setPreventClose(true);
      });

      // Register Cmd+W hotkey on macOS to close window
      if (Util.isMacos()) {
        await hotKeyManager.unregisterAll();
        HotKey hotKey = HotKey(
          key: PhysicalKeyboardKey.keyW,
          modifiers: [HotKeyModifier.meta],
          scope: HotKeyScope.inapp,
        );
        await hotKeyManager.register(
          hotKey,
          keyDownHandler: (hotKey) {
            windowManager.hide();
          },
        );
      }
    }
  }

  initLogger();

  try {
    await initPackageInfo();
  } catch (e) {
    logger.e("init package info fail", e);
  }

  if (args.isDownloadPopup) {
    api.init(
      args.popupNetworkValue,
      args.popupAddressValue,
      args.popupApiTokenValue,
    );
    return;
  }

  final controller =
      Get.put(AppController(hiddenFromArgs: args.hiddenFromArgs));
  try {
    await controller.loadStartConfig();
    final startCfg = controller.startConfig.value;
    controller.runningPort.value = await LibgopeedBoot.instance.start(startCfg);
    api.init(startCfg.network, controller.runningAddress(), startCfg.apiToken);
  } catch (e) {
    logger.e("libgopeed init fail", e);
  }

  try {
    await controller.loadDownloaderConfig();
  } catch (e) {
    logger.e("load config fail", e);
  }

  // Auto-start incomplete tasks if enabled
  if (controller.downloaderConfig.value.extra.autoStartTasks) {
    try {
      await api.continueAllTasks(null);
      logger.i("auto-start tasks completed");
    } catch (e) {
      logger.w("auto-start tasks fail", e);
    }
  }

  () async {
    if (Util.isDesktop()) {
      try {
        registerUrlScheme("gopeed");
        if (controller.downloaderConfig.value.extra.defaultBtClient) {
          registerDefaultTorrentClient();
        }
      } catch (e) {
        logger.e("register scheme fail", e);
      }

      try {
        await installHost();
      } catch (e) {
        logger.e("browser extension host binary install fail", e);
      }
      for (final browser in Browser.values) {
        try {
          await installManifest(browser);
        } catch (e) {
          logger.e(
              "browser [${browser.name}] extension host integration fail", e);
        }
      }

      try {
        await installUpdater();
      } catch (e) {
        logger.e("updater install fail", e);
      }
    }
  }();
}

Future<void> onStart() async {
  // if is debug mode, check language message is complete,change debug locale to your comfortable language if you want
  if (kDebugMode) {
    final debugLang = getLocaleKey(debugLocale);
    final fullMessages = messages.keys[debugLang];
    messages.keys.keys.where((e) => e != debugLang).forEach((lang) {
      final langMessages = messages.keys[lang];
      if (langMessages == null) {
        logger.w("missing language: $lang");
        return;
      }
      final missingKeys =
          fullMessages!.keys.where((key) => langMessages[key] == null);
      if (missingKeys.isNotEmpty) {
        logger.w("missing language: $lang, keys: $missingKeys");
      }
    });
  }

  if (Config.isConfigured && Database.instance.getAnalyticsEnabled()) {
    try {
      await Analytics.instance.init();
      await Analytics.instance.logAppOpen();
    } catch (e) {
      logger.w("GA4 init failed: $e");
    }
  }
}
