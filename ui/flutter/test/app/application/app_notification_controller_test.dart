import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
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
    final notifications = calls.where((call) => call.method == 'show').toList();
    expect(notifications, hasLength(2));
    expect(notifications.map((call) => (call.arguments as Map)['body']), ['archive.zip', 'failed.zip']);

    runtime.setNotificationsEnabled(false);
    await container.pump();
    await container.read(appNotificationControllerProvider.future);
    events.add(done);
    await Future<void>.delayed(Duration.zero);
    expect(calls.where((call) => call.method == 'show'), hasLength(2));
  });
}

class _TestNotificationController extends AppNotificationController {
  _TestNotificationController(this.taskEvents);

  @override
  final Stream<TaskEvent> taskEvents;
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
