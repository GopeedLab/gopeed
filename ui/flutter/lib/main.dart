import 'package:args/args.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:get/get.dart';
import 'package:hotkey_manager/hotkey_manager.dart';
import 'package:window_manager/window_manager.dart';

import 'api/api.dart' as api;
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

class Args {
  static const flagHidden = "hidden";

  bool hidden = false;

  Args();

  Args.parse(List<String> args) {
    final parser = ArgParser();
    parser.addFlag(flagHidden);
    final results = parser.parse(args);
    hidden = results.flag(flagHidden);
  }
}

void main(List<String> arguments) async {
  Args args;
  // Parse url scheme arguments, e.g. gopeed:?hidden=true
  // TODO: macos open url handle
  // TODO: macos updater test
  if (arguments.firstOrNull?.startsWith("gopeed:") == true) {
    try {
      final uri = Uri.parse(arguments.first);
      args = Args()..hidden = uri.queryParameters["hidden"] == "true";
    } catch (e) {
      // ignore
      args = Args.parse([]);
    }
  } else {
    args = Args.parse(arguments);
  }

  await init(args);
  onStart();

  runApp(const AppView());
}

Future<void> init(Args args) async {
  WidgetsFlutterBinding.ensureInitialized();
  if (Util.isMobile()) {
    FlutterForegroundTask.initCommunicationPort();
  }
  await Util.initStorageDir();
  await Database.instance.init();
  if (Util.isDesktop()) {
    await windowManager.ensureInitialized();
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
      if (!args.hidden) {
        await windowManager.show();
        await windowManager.focus();
      }
      await windowManager.setPreventClose(true);
      // windows_manager has a bug where when window to be maximized, it will be unmaximized immediately, so can't implement this feature currently.
      // https://github.com/leanflutter/window_manager/issues/412
      // if (windowState.isMaximized) {
      //   await windowManager.maximize();
      // }
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

  initLogger();

  try {
    await initPackageInfo();
  } catch (e) {
    logger.e("init package info fail", e);
  }

  final controller = Get.put(AppController());
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
}
