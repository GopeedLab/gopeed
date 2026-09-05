import 'dart:async';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';

import '../../api/model/task.dart';
import '../../core/capabilities/app_capabilities.dart';
import '../../l10n/l10n.dart';
import '../../util/util.dart';
import 'app_runtime_controller.dart';

final appNotificationControllerProvider = AsyncNotifierProvider<AppNotificationController, AppNotificationState>(
  AppNotificationController.new,
);

class AppNotificationState {
  const AppNotificationState({this.started = false});

  final bool started;
}

class AppNotificationController extends AsyncNotifier<AppNotificationState> {
  final FlutterLocalNotificationsPlugin _plugin = FlutterLocalNotificationsPlugin();
  final Map<String, Status> _previousStatus = {};
  Timer? _timer;
  var _notificationId = 0;

  @override
  Future<AppNotificationState> build() async {
    final runtime = ref.watch(appRuntimeControllerProvider).value;
    if (runtime == null || kIsWeb || !Util.isDesktop()) {
      return const AppNotificationState();
    }
    await _initNotifications(appLocalizationsFor(runtime.downloaderConfig.extra.locale));
    _startPolling();
    ref.onDispose(() {
      _timer?.cancel();
    });
    return const AppNotificationState(started: true);
  }

  Future<void> _initNotifications(AppLocalizations locale) async {
    const darwin = DarwinInitializationSettings(
      requestAlertPermission: false,
      requestBadgePermission: false,
      requestSoundPermission: false,
    );
    final linux = LinuxInitializationSettings(
      defaultActionName: locale.open,
      defaultIcon: AssetsLinuxIcon('assets/icon/icon.png'),
    );

    String? windowsIconPath;
    try {
      if (Util.isWindows()) {
        final byteData = await rootBundle.load('assets/icon/icon.ico');
        final tempDir = await getTemporaryDirectory();
        final file = File('${tempDir.path}/notification_icon.ico');
        await file.writeAsBytes(byteData.buffer.asUint8List(byteData.offsetInBytes, byteData.lengthInBytes));
        windowsIconPath = file.path;
      }
    } catch (_) {}

    final windows = WindowsInitializationSettings(
      appName: 'Gopeed',
      appUserModelId: 'com.gopeed.gopeed',
      guid: '3c1bf3f4-3d91-4eaa-a33f-8705e71cf1ce',
      iconPath: windowsIconPath,
    );

    await _plugin.initialize(
      settings: InitializationSettings(macOS: darwin, linux: linux, windows: windows),
    );
  }

  void _startPolling() {
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 2), (_) async {
      try {
        final runtime = ref.read(appRuntimeControllerProvider).value;
        if (runtime?.downloaderConfig.extra.desktopNotification == false) {
          return;
        }
        final tasks = await ref.read(gopeedServiceProvider).getTasks(Status.values);
        final locale = appLocalizationsFor(runtime?.downloaderConfig.extra.locale ?? '');
        for (final task in tasks) {
          final previous = _previousStatus[task.id];
          if (previous != null && previous != task.status) {
            if (task.status == Status.done) {
              await _showNotification(title: locale.notificationTaskDone, body: task.name);
            } else if (task.status == Status.error) {
              await _showNotification(title: locale.notificationTaskError, body: task.name);
            }
          }
          _previousStatus[task.id] = task.status;
        }
        final currentIds = tasks.map((task) => task.id).toSet();
        _previousStatus.removeWhere((id, _) => !currentIds.contains(id));
      } catch (_) {}
    });
  }

  Future<void> _showNotification({required String title, required String body}) async {
    const details = NotificationDetails(
      macOS: DarwinNotificationDetails(),
      linux: LinuxNotificationDetails(),
      windows: WindowsNotificationDetails(),
    );
    await _plugin.show(id: _notificationId++, title: title, body: body, notificationDetails: details);
  }
}
