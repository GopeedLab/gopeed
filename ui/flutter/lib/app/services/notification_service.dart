import 'dart:async';
import 'dart:io';

import 'package:flutter/services.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:get/get.dart';
import 'package:path_provider/path_provider.dart';

import '../../api/api.dart';
import '../../api/model/task.dart';
import '../../util/util.dart';
import '../modules/app/controllers/app_controller.dart';

class NotificationService extends GetxService {
  Timer? _timer;
  final Map<String, Status> _previousStatus = {};

  final AppController appController = Get.find<AppController>();
  final FlutterLocalNotificationsPlugin flutterLocalNotificationsPlugin =
      FlutterLocalNotificationsPlugin();

  @override
  void onInit() {
    super.onInit();
    _initNotifications();
    _startPolling();
  }

  Future<void> _initNotifications() async {
    const DarwinInitializationSettings initializationSettingsDarwin =
        DarwinInitializationSettings(
            requestAlertPermission: false,
            requestBadgePermission: false,
            requestSoundPermission: false);

    final LinuxInitializationSettings initializationSettingsLinux =
        LinuxInitializationSettings(
      defaultActionName: 'Open notification',
      defaultIcon: AssetsLinuxIcon('assets/icon/icon.png'),
    );

    String? windowsIconPath;
    try {
      if (Util.isWindows()) {
        final byteData = await rootBundle.load('assets/icon/icon.ico');
        final tempDir = await getTemporaryDirectory();
        final file = File('${tempDir.path}/notification_icon.ico');
        await file.writeAsBytes(byteData.buffer
            .asUint8List(byteData.offsetInBytes, byteData.lengthInBytes));
        windowsIconPath = file.path;
      }
    } catch (e) {
      // Ignore
    }

    final WindowsInitializationSettings initializationSettingsWindows =
        WindowsInitializationSettings(
      appName: 'Gopeed',
      appUserModelId: 'com.gopeed.gopeed',
      guid: '3c1bf3f4-3d91-4eaa-a33f-8705e71cf1ce', // unique guid
      iconPath: windowsIconPath,
    );

    final InitializationSettings initializationSettings =
        InitializationSettings(
      macOS: initializationSettingsDarwin,
      linux: initializationSettingsLinux,
      windows: initializationSettingsWindows,
    );

    await flutterLocalNotificationsPlugin.initialize(
      settings: initializationSettings,
    );
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

  int _notificationId = 0;

  Future<void> _showNotification(
      {required String title, required String body}) async {
    const NotificationDetails notificationDetails = NotificationDetails(
      macOS: DarwinNotificationDetails(),
      linux: LinuxNotificationDetails(),
      windows: WindowsNotificationDetails(),
    );

    await flutterLocalNotificationsPlugin.show(
      id: _notificationId++,
      title: title,
      body: body,
      notificationDetails: notificationDetails,
    );
  }
}
