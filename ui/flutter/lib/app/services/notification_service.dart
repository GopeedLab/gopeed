import 'dart:async';

import 'package:get/get.dart';
import 'package:local_notifier/local_notifier.dart';

import 'package:uri_to_file/uri_to_file.dart';
import 'package:path_provider/path_provider.dart';
import 'dart:io';
import 'package:flutter/services.dart';

import '../../api/api.dart';
import '../../api/model/task.dart';
import '../../util/util.dart';
import '../modules/app/controllers/app_controller.dart';

class NotificationService extends GetxService {
  Timer? _timer;
  final Map<String, Status> _previousStatus = {};

  final AppController appController = Get.find<AppController>();

  @override
  void onInit() {
    super.onInit();
    _startPolling();
  }

  @override
  void onClose() {
    _timer?.cancel();
    super.onClose();
  }

  void _startPolling() {
    _timer = Timer.periodic(const Duration(seconds: 2), (timer) async {
      try {
        final config = appController.downloaderConfig.value;
        if (!config.extra.desktopNotification) {
          return;
        }

        final tasks = await getTasks([
          Status.ready,
          Status.running,
          Status.pause,
          Status.wait,
          Status.error,
          Status.done,
        ]);

        for (var task in tasks) {
          final prevStatus = _previousStatus[task.id];
          final currentStatus = task.status;

          if (prevStatus != null && prevStatus != currentStatus) {
            if (currentStatus == Status.done) {
              _showNotification(
                title: 'notificationTaskDone'.tr,
                body: task.name,
              );
            } else if (currentStatus == Status.error) {
              _showNotification(
                title: 'notificationTaskError'.tr,
                body: task.name,
              );
            }
          }
          _previousStatus[task.id] = currentStatus;
        }

        // Clean up deleted tasks from map
        final currentTaskIds = tasks.map((t) => t.id).toSet();
        _previousStatus
            .removeWhere((id, status) => !currentTaskIds.contains(id));
      } catch (e) {
        // Ignored
      }
    });
  }

  Future<void> _showNotification(
      {required String title, required String body}) async {
    String? imagePath;

    try {
      if (Util.isWindows() || Util.isLinux()) {
        final assetPath =
            Util.isWindows() ? 'assets/icon/icon.ico' : 'assets/icon/icon.png';
        final byteData = await rootBundle.load(assetPath);
        final tempDir = await getTemporaryDirectory();
        final file = File('${tempDir.path}/${assetPath.split('/').last}');
        await file.writeAsBytes(byteData.buffer
            .asUint8List(byteData.offsetInBytes, byteData.lengthInBytes));
        imagePath = file.path;
      }
    } catch (e) {
      // Ignore if icon extraction fails
    }

    final notification = LocalNotification(
      title: title,
      body: body,
      imagePath: imagePath,
    );
    notification.show();
  }
}
