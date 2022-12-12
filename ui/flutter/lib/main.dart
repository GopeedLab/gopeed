import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/i18n/messages.dart';

import 'core/libgopeed_boot.dart';
import 'pages/app/app_controller.dart';
import 'pages/app/app_view.dart';
import 'util/log_util.dart';
import 'util/mac_secure_util.dart';
import 'util/package_info.dart';

void main() async {
  await init();
  onStart();

  runApp(const AppView());
}

Future<void> init() async {
  WidgetsFlutterBinding.ensureInitialized();

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
  final appController = Get.find<AppController>();
  await appController.trackerUpdateOnStart();

  // if is debug mode, check language message is complete
  if (kDebugMode) {
    final mainLang = getLocaleKey(mainLocale);
    final fullMessages = messages.keys[mainLang];
    availableLanguages.where((e) => e != mainLang).forEach((lang) {
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
