import 'dart:async';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/api/model/meta.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/task.dart' as api_task;
import 'package:gopeed/app/application/app_notification_controller.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/core/common/api_server_state.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:gopeed/core/common/task_event.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('dexterous.com/flutter/local_notifications');
  final calls = <MethodCall>[];
  var permissionGranted = true;
  late StreamController<TaskEvent> events;

  setUp(() {
    debugDefaultTargetPlatformOverride = TargetPlatform.macOS;
    MacOSFlutterLocalNotificationsPlugin.registerWith();
    calls.clear();
    permissionGranted = true;
    events = StreamController<TaskEvent>.broadcast();
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(channel, (call) async {
      calls.add(call);
      return permissionGranted;
    });
  });

  tearDown(() {
    unawaited(events.close());
    debugDefaultTargetPlatformOverride = null;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(channel, null);
  });

  Future<ProviderContainer> start({required bool enabled}) async {
    final container = ProviderContainer(
      overrides: [
        appRuntimeControllerProvider.overrideWith(() => _TestRuntimeController(enabled)),
        appNotificationControllerProvider.overrideWith(() => _TestNotificationController(events.stream)),
      ],
    );
    addTearDown(container.dispose);
    await container.read(appRuntimeControllerProvider.future);
    await container.read(appNotificationControllerProvider.future);
    return container;
  }

  Map<dynamic, dynamic> initialization() => calls.lastWhere((call) => call.method == 'initialize').arguments as Map;

  List<Map<dynamic, dynamic>> shownNotifications() =>
      calls.where((call) => call.method == 'show').map((call) => call.arguments as Map).toList();

  test('macOS requests alerts and sound when download notifications are enabled at startup', () async {
    await start(enabled: true);

    expect(initialization()['requestAlertPermission'], isTrue);
    expect(initialization()['requestSoundPermission'], isTrue);
    expect(initialization()['requestBadgePermission'], isFalse);
  });

  test('disabled notifications defer authorization until the setting is enabled', () async {
    final container = await start(enabled: false);
    expect(initialization()['requestAlertPermission'], isFalse);
    expect(initialization()['requestSoundPermission'], isFalse);

    final runtime = container.read(appRuntimeControllerProvider.notifier) as _TestRuntimeController;
    runtime.setNotificationsEnabled(true);
    await container.pump();
    await container.read(appNotificationControllerProvider.future);

    expect(initialization()['requestAlertPermission'], isTrue);
    expect(initialization()['requestSoundPermission'], isTrue);

    runtime.setNotificationsEnabled(false);
    await container.pump();
    await container.read(appNotificationControllerProvider.future);
    expect(initialization()['requestAlertPermission'], isFalse);
    expect(initialization()['requestSoundPermission'], isFalse);
  });

  test('denied macOS authorization does not fail notification controller startup', () async {
    permissionGranted = false;
    final container = await start(enabled: true);

    expect(container.read(appNotificationControllerProvider).requireValue.started, isTrue);
  });

  test('terminal task events notify once after enabling and stop after disabling', () async {
    final container = await start(enabled: false);
    final runtime = container.read(appRuntimeControllerProvider.notifier) as _TestRuntimeController;
    const done = TaskEvent(type: TaskEventType.done, taskId: '1', name: 'archive.zip');
    events.add(done);
    await Future<void>.delayed(Duration.zero);
    expect(calls.where((call) => call.method == 'show'), isEmpty);

    runtime.setNotificationsEnabled(true);
    await container.pump();
    await container.read(appNotificationControllerProvider.future);
    events.add(done);
    events.add(const TaskEvent(type: TaskEventType.error, taskId: '2', name: 'failed.zip'));
    await Future<void>.delayed(Duration.zero);
    final notifications = shownNotifications();
    expect(notifications, hasLength(2));
    expect(notifications.map((call) => call['body']), ['archive.zip', 'failed.zip']);

    runtime.setNotificationsEnabled(false);
    await container.pump();
    await container.read(appNotificationControllerProvider.future);
    events.add(done);
    await Future<void>.delayed(Duration.zero);
    expect(calls.where((call) => call.method == 'show'), hasLength(2));
  });

  test('macOS registers action categories for done task combinations', () async {
    await start(enabled: true);

    final categories = initialization()['notificationCategories'] as List<dynamic>;
    final identifiers = categories.map((category) => (category as Map)['identifier']).toList();
    expect(identifiers, ['taskDoneSingleFile', 'taskDoneFolder']);
    final singleFile = categories.firstWhere((c) => c['identifier'] == 'taskDoneSingleFile') as Map;
    final singleActions = (singleFile['actions'] as List).cast<Map>();
    expect(singleActions.map((action) => action['identifier']), ['open_file', 'open_folder']);
    expect(singleActions.map((action) => action['title']), ['Open File', 'Open Folder']);
  });

  test('a done single-file task resolves an open-file target', () async {
    final container = await start(enabled: true);
    final notifier = container.read(appNotificationControllerProvider.notifier) as _TestNotificationController;

    final target = await notifier.resolveTaskTarget('single');

    expect(target?.path, '/downloads/archive.zip');
    expect(target?.canOpenFile, isTrue);
  });

  test('a folder task never resolves an open-file target', () async {
    final container = await start(enabled: true);
    final notifier = container.read(appNotificationControllerProvider.notifier) as _TestNotificationController;

    final target = await notifier.resolveTaskTarget('folder');

    expect(target?.path, '/downloads/torrent-bundle');
    expect(target?.canOpenFile, isFalse);
  });

  test('a done single-file task notification selects the single-file category and payload path', () async {
    await start(enabled: true);
    events.add(const TaskEvent(type: TaskEventType.done, taskId: 'single', name: 'archive.zip'));
    await Future<void>.delayed(Duration.zero);

    final notifications = shownNotifications();
    expect(notifications, hasLength(1));
    final platformSpecifics = notifications[0]['platformSpecifics'] as Map<dynamic, dynamic>;
    expect(platformSpecifics['categoryIdentifier'], 'taskDoneSingleFile');
    expect(notifications[0]['payload'], '{"path":"/downloads/archive.zip"}');
  });

  test('a done folder task notification selects the folder category', () async {
    await start(enabled: true);
    events.add(const TaskEvent(type: TaskEventType.done, taskId: 'folder', name: 'torrent-bundle'));
    await Future<void>.delayed(Duration.zero);

    final platformSpecifics = shownNotifications()[0]['platformSpecifics'] as Map<dynamic, dynamic>;
    expect(platformSpecifics['categoryIdentifier'], 'taskDoneFolder');
  });

  test('an errored task notification carries no actions at all', () async {
    await start(enabled: true);
    events.add(const TaskEvent(type: TaskEventType.error, taskId: 'single', name: 'archive.zip'));
    await Future<void>.delayed(Duration.zero);

    final notifications = shownNotifications();
    expect(notifications, hasLength(1));
    final platformSpecifics = notifications[0]['platformSpecifics'] as Map<dynamic, dynamic>;
    expect(platformSpecifics['categoryIdentifier'], isNull);
    expect(notifications[0]['payload'], '');
  });

  test('action responses route to file and folder handlers by encoded spec', () async {
    final container = await start(enabled: true);
    final notifier = container.read(appNotificationControllerProvider.notifier) as _TestNotificationController;
    final existingPath = '${Directory.systemTemp.path}/gopeed-notification-test.zip';
    addTearDown(() => File(existingPath).deleteSync());
    File(existingPath).writeAsStringSync('data');

    // Windows style: the action arguments carry the full spec.
    await notifier.handleNotificationResponse(
      NotificationResponse(
        notificationResponseType: NotificationResponseType.selectedNotificationAction,
        actionId: NotificationActionSpec(id: 'open_file', path: existingPath).encode(),
      ),
    );
    expect(notifier.openedFiles, [existingPath]);
    expect(notifier.revealedFolders, isEmpty);
    expect(notifier.frontedWindow, isFalse);

    // macOS/Linux style: stable action key plus the notification payload path.
    await notifier.handleNotificationResponse(
      NotificationResponse(
        notificationResponseType: NotificationResponseType.selectedNotificationAction,
        actionId: 'open_folder',
        payload: '{"path":"$existingPath"}',
      ),
    );
    expect(notifier.revealedFolders, [existingPath]);
    expect(notifier.frontedWindow, isFalse);
  });

  test('a plain notification body click brings the app window to the front', () async {
    final container = await start(enabled: true);
    final notifier = container.read(appNotificationControllerProvider.notifier) as _TestNotificationController;

    await notifier.handleNotificationResponse(
      const NotificationResponse(notificationResponseType: NotificationResponseType.selectedNotification),
    );

    expect(notifier.frontedWindow, isTrue);
    expect(notifier.openedFiles, isEmpty);
    expect(notifier.revealedFolders, isEmpty);
  });
}

api_task.Task _task(String id, {required bool folder}) {
  final task = api_task.Task(
    id: id,
    name: folder ? 'torrent-bundle' : 'archive.zip',
    meta: Meta(
      req: Request(url: 'https://example.com/archive.zip'),
      opts: Options(path: '/downloads'),
    ),
    status: api_task.Status.done,
    uploading: false,
    progress: api_task.Progress(used: 0, speed: 0, downloaded: 1, uploadSpeed: 0, uploaded: 0),
    createdAt: DateTime(2026),
    updatedAt: DateTime(2026),
  );
  task.meta.res = Resource(name: folder ? 'torrent-bundle' : '', files: const []);
  return task;
}

class _TestNotificationController extends AppNotificationController {
  _TestNotificationController(this.taskEvents);

  @override
  final Stream<TaskEvent> taskEvents;

  final tasksById = {
    'single': _task('single', folder: false),
    'folder': _task('folder', folder: true),
  };
  final openedFiles = <String>[];
  final revealedFolders = <String>[];
  var frontedWindow = false;

  @override
  Future<api_task.Task?> findTask(String taskId) async => tasksById[taskId];

  @override
  Future<bool> openTaskFile(String filePath) async {
    openedFiles.add(filePath);
    return true;
  }

  @override
  Future<bool> revealTaskFolder(String filePath) async {
    revealedFolders.add(filePath);
    return true;
  }

  @override
  Future<void> bringWindowToFront() async {
    frontedWindow = true;
  }
}

class _TestRuntimeController extends AppRuntimeController {
  _TestRuntimeController(this.enabled);

  final bool enabled;

  @override
  Future<AppRuntimeState> build() async => _runtime(enabled);

  void setNotificationsEnabled(bool enabled) => state = AsyncData(_runtime(enabled));

  AppRuntimeState _runtime(bool enabled) => AppRuntimeState(
    startConfig: StartConfig(),
    apiServerState: ApiServerState.fromJson({}),
    downloaderConfig: DownloaderConfig()..extra.desktopNotification = enabled,
  );
}
