import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:get/get.dart';
import 'package:window_manager/window_manager.dart';

import 'api/api.dart' as api;
import 'app/modules/app/controllers/app_controller.dart';
import 'app/modules/app/views/app_view.dart';
import 'core/libgopeed_boot.dart';
import 'database/database.dart';
import 'database/entity.dart';
import 'i18n/message.dart';
import 'util/locale_manager.dart';
import 'util/log_util.dart';
import 'util/mac_secure_util.dart';
import 'util/package_info.dart';
import 'util/util.dart';

void main() async {
  await init();
  onStart();

  runApp(const AppView());
}

Future<void> init() async {
  WidgetsFlutterBinding.ensureInitialized();
  await Util.initStorageDir();
  await Database.instance.init();
  if (Util.isDesktop()) {
    await windowManager.ensureInitialized();
    final windowState = Database.instance.getWindowState();
    final windowOptions = WindowOptions(
      size: Size(windowState?.width ?? 800, windowState?.height ?? 600),
      center: true,
      skipTaskbar: false,
    );
    await windowManager.waitUntilReadyToShow(windowOptions, () async {
      await windowManager.show();
      await windowManager.focus();
      await windowManager.setPreventClose(true);
      // windows_manager has a bug where when window to be maximized, it will be unmaximized immediately, so can't implement this feature currently.
      // https://github.com/leanflutter/window_manager/issues/412
      // if (windowState.isMaximized) {
      //   await windowManager.maximize();
      // }
    });
  }

  initLogger();

  final controller = Get.put(AppController());
  try {
    await controller.loadStartConfig();
    final startCfg = controller.startConfig.value;
    // When the app is closed by swiping on the mobile, the backend server is actually still running, don't need to start again.
    final isRunning =
        Util.isMobile() && (await FlutterForegroundTask.isRunningService);
    if (!isRunning) {
      controller.runningPort.value =
          await LibgopeedBoot.instance.start(startCfg);
      api.init(
          startCfg.network, controller.runningAddress(), startCfg.apiToken);
      Database.instance.saveLastRunningConfig(StartConfigEntity(
          network: startCfg.network,
          address: controller.runningAddress(),
          apiToken: startCfg.apiToken));
    } else {
      final lastRunningConfig = Database.instance.getLastRunningConfig()!;
      api.init(lastRunningConfig.network, lastRunningConfig.address,
          lastRunningConfig.apiToken);
    }
  } catch (e) {
    logger.e("libgopeed init fail", e);
  }

  try {
    await controller.loadDownloaderConfig();
    MacSecureUtil.loadBookmark();
  } catch (e) {
    logger.e("load config fail", e);
  }

  try {
    await initPackageInfo();
  } catch (e) {
    logger.e("init package info fail", e);
  }
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
