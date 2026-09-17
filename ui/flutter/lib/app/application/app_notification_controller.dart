import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path/path.dart' as path;
import 'package:window_manager/window_manager.dart';

import '../../api/model/task.dart' as api_task;
import '../../core/capabilities/app_capabilities.dart';
import '../../core/common/task_event.dart';
import '../../core/libgopeed_boot.dart';
import '../../core/utils/file_explorer.dart';
import '../../features/tasks/domain/task_record.dart';
import '../../l10n/l10n.dart';
import '../../util/log_util.dart';
import '../../util/util.dart';
import 'app_runtime_controller.dart';

final appNotificationControllerProvider = AsyncNotifierProvider<AppNotificationController, AppNotificationState>(
  AppNotificationController.new,
);

const _openFileAction = 'open_file';
const _openFolderAction = 'open_folder';

// macOS resolves buttons through pre-registered notification categories, one
// per action combination a done task event can offer. Error notifications
// carry no buttons at all.
const _categoryDoneSingleFile = 'taskDoneSingleFile';
const _categoryDoneFolder = 'taskDoneFolder';

/// A notification button resolved for one task event.
///
/// [id] is one of [_openFileAction]/[_openFolderAction]; Windows carries
/// [path] inside the action arguments, macOS and Linux deliver it through the
/// notification payload, so the click callback can act without looking the
/// task up again (it may already be gone by the time the user clicks).
class NotificationActionSpec {
  const NotificationActionSpec({required this.id, required this.path});

  final String id;
  final String path;

  String encode() => jsonEncode({'action': id, 'path': path});

  static NotificationActionSpec? decode(String raw) {
    if (raw.isEmpty) return null;
    try {
      final data = jsonDecode(raw);
      if (data is! Map<String, dynamic>) return null;
      final id = data['action'];
      final targetPath = data['path'];
      if (id is! String || targetPath is! String) return null;
      if (id != _openFileAction && id != _openFolderAction) return null;
      return NotificationActionSpec(id: id, path: targetPath);
    } catch (_) {
      return null;
    }
  }
}

class AppNotificationState {
  const AppNotificationState({this.started = false});

  final bool started;
}

class AppNotificationController extends AsyncNotifier<AppNotificationState> {
  final FlutterLocalNotificationsPlugin _plugin = FlutterLocalNotificationsPlugin();
  StreamSubscription<TaskEvent>? _taskEventSubscription;
  var _notificationId = 0;

  @visibleForTesting
  Stream<TaskEvent> get taskEvents => LibgopeedBoot.instance.taskEvents;

  @override
  Future<AppNotificationState> build() async {
    final runtime = ref.watch(appRuntimeControllerProvider).value;
    if (runtime == null || kIsWeb || !Util.isDesktop()) {
      return const AppNotificationState();
    }
    await _initNotifications(
      appLocalizationsFor(runtime.downloaderConfig.extra.locale),
      requestPermissions: runtime.downloaderConfig.extra.desktopNotification,
    );
    _listenTaskEvents();
    ref.onDispose(() {
      unawaited(_taskEventSubscription?.cancel());
    });
    return const AppNotificationState(started: true);
  }

  Future<void> _initNotifications(AppLocalizations locale, {required bool requestPermissions}) async {
    // The runtime is watched, so enabling notifications in settings also
    // requests authorization. macOS remembers previously granted/denied access.
    final openFileAction = DarwinNotificationAction.plain(_openFileAction, locale.openFile);
    final openFolderAction = DarwinNotificationAction.plain(_openFolderAction, locale.openFolder);
    final darwin = DarwinInitializationSettings(
      requestAlertPermission: requestPermissions,
      requestBadgePermission: false,
      requestSoundPermission: requestPermissions,
      notificationCategories: [
        DarwinNotificationCategory(_categoryDoneSingleFile, actions: [openFileAction, openFolderAction]),
        DarwinNotificationCategory(_categoryDoneFolder, actions: [openFolderAction]),
      ],
    );
    final linux = LinuxInitializationSettings(
      defaultActionName: locale.open,
      defaultIcon: AssetsLinuxIcon('assets/icon/icon.png'),
    );

    String? windowsIconPath;
    try {
      if (Util.isWindows()) {
        // Windows requires a file path for IconUri, so use the bundled PNG
        // beside the executable rather than copying it to another directory.
        final file = File(
          path.join(
            path.dirname(Platform.resolvedExecutable),
            'data',
            'flutter_assets',
            'assets',
            'icon',
            'icon_512.png',
          ),
        );
        if (await file.exists()) {
          windowsIconPath = file.absolute.path;
        } else {
          logger.w('Windows notification icon not found: ${file.path}');
        }
      }
    } catch (error, stackTrace) {
      logger.w('prepare Windows notification icon failed', error, stackTrace);
    }

    final windows = WindowsInitializationSettings(
      appName: 'Gopeed',
      appUserModelId: 'com.gopeed.gopeed',
      guid: '3c1bf3f4-3d91-4eaa-a33f-8705e71cf1ce',
      iconPath: windowsIconPath,
    );

    await _plugin.initialize(
      settings: InitializationSettings(macOS: darwin, linux: linux, windows: windows),
      onDidReceiveNotificationResponse: _onNotificationResponse,
    );
  }

  void _listenTaskEvents() {
    _taskEventSubscription?.cancel();
    _taskEventSubscription = taskEvents.listen((event) async {
      final runtime = ref.read(appRuntimeControllerProvider).value;
      if (runtime?.downloaderConfig.extra.desktopNotification == false) return;
      final locale = appLocalizationsFor(runtime?.downloaderConfig.extra.locale ?? '');
      switch (event.type) {
        case TaskEventType.done:
          await _showNotification(
            locale: locale,
            title: locale.notificationTaskDone,
            body: event.name,
            taskId: event.taskId,
            withActions: true,
          );
        case TaskEventType.error:
          // A failed download has no usable output to open, so no buttons.
          await _showNotification(
            locale: locale,
            title: locale.notificationTaskError,
            body: event.name,
            taskId: event.taskId,
            withActions: false,
          );
      }
    });
  }

  Future<void> _showNotification({
    required AppLocalizations locale,
    required String title,
    required String body,
    required String taskId,
    required bool withActions,
  }) async {
    final target = withActions ? await resolveTaskTarget(taskId) : null;
    final payload = target == null ? '' : jsonEncode({'path': target.path});
    final details = NotificationDetails(
      macOS: DarwinNotificationDetails(
        categoryIdentifier: switch (target) {
          null => null,
          _ when target.canOpenFile => _categoryDoneSingleFile,
          _ => _categoryDoneFolder,
        },
      ),
      linux: LinuxNotificationDetails(
        actions: [
          if (target != null && target.canOpenFile)
            LinuxNotificationAction(key: _openFileAction, label: locale.openFile),
          if (target != null) LinuxNotificationAction(key: _openFolderAction, label: locale.openFolder),
        ],
      ),
      windows: WindowsNotificationDetails(
        actions: [
          if (target != null && target.canOpenFile)
            WindowsAction(content: locale.openFile, arguments: NotificationActionSpec(id: _openFileAction, path: target.path).encode()),
          if (target != null)
            WindowsAction(content: locale.openFolder, arguments: NotificationActionSpec(id: _openFolderAction, path: target.path).encode()),
        ],
      ),
    );
    await _plugin.show(id: _notificationId++, title: title, body: body, notificationDetails: details, payload: payload);
  }

  /// Resolves the download target a done task event points at, following the
  /// same rule as the task details page: only single-file tasks offer "open
  /// file", folder tasks (e.g. multi-file torrents) can only reveal the
  /// folder. Returns null when the task is already gone.
  @visibleForTesting
  Future<({String path, bool canOpenFile})?> resolveTaskTarget(String taskId) async {
    final apiTask = await findTask(taskId);
    if (apiTask == null) return null;
    final record = TaskRecord.fromApi(apiTask);
    return (path: record.storagePath, canOpenFile: !record.isFolder);
  }

  @visibleForTesting
  Future<void> handleNotificationResponse(NotificationResponse response) => _onNotificationResponse(response);

  Future<void> _onNotificationResponse(NotificationResponse response) async {
    // Windows actions carry the full spec in their arguments.
    var spec = NotificationActionSpec.decode(response.actionId ?? '');
    if (spec == null) {
      // macOS/Linux actions are stable keys; the path arrives in the payload.
      final id = response.actionId ?? '';
      if (id == _openFileAction || id == _openFolderAction) {
        try {
          final data = jsonDecode(response.payload ?? '');
          final targetPath = data is Map<String, dynamic> ? data['path'] : null;
          if (targetPath is String) {
            spec = NotificationActionSpec(id: id, path: targetPath);
          }
        } catch (_) {
          // Malformed payload: fall through to the plain-click behavior.
        }
      }
    }
    if (spec == null) {
      // Plain click on the notification body: bring the app window back.
      await bringWindowToFront();
      return;
    }
    // Use the task card's file operations, including reveal's parent fallback.
    switch (spec.id) {
      case _openFileAction:
        if (!await openTaskFile(spec.path)) {
          logger.w('failed to open file from notification: ${spec.path}');
        }
      case _openFolderAction:
        if (!await revealTaskFolder(spec.path)) {
          logger.w('failed to reveal folder from notification: ${spec.path}');
        }
    }
  }

  @visibleForTesting
  Future<api_task.Task?> findTask(String taskId) async {
    try {
      final tasks = await ref.read(gopeedServiceProvider).getTasks(api_task.Status.values);
      for (final task in tasks) {
        if (task.id == taskId) return task;
      }
    } catch (error, stackTrace) {
      logger.w('look up task for notification failed', error, stackTrace);
    }
    return null;
  }

  @visibleForTesting
  Future<bool> openTaskFile(String filePath) => FileExplorer.open(filePath);

  @visibleForTesting
  Future<bool> revealTaskFolder(String filePath) => FileExplorer.reveal(filePath);

  @visibleForTesting
  Future<void> bringWindowToFront() async {
    await windowManager.show();
    await windowManager.focus();
  }
}
