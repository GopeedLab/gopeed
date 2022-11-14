import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/api/api.dart' as api;

import 'core/libgopeed_boot.dart';
import 'pages/app/app_controller.dart';
import 'pages/app/app_view.dart';
import 'util/log_util.dart';
import 'util/mac_secure_util.dart';

void main() async {
  await init();

  runApp(const AppView());
}

Future<void> init() async {
  WidgetsFlutterBinding.ensureInitialized();

  final controller = Get.put(AppController());
  try {
    await controller.loadStartConfig();
    final startCfg = controller.startConfig.value;
    startCfg.runningPort =
        await LibgopeedBoot.instance.start(startCfg.network, startCfg.address);
    api.init(startCfg.network, startCfg.runningAddress);
  } catch (e) {
    logger.e("libgopeed init fail", e);
  }

  try {
    await controller.loadDownloaderConfig();
    MacSecureUtil.loadBookmark();
  } catch (e) {
    logger.e("load config fail", e);
  }
}
