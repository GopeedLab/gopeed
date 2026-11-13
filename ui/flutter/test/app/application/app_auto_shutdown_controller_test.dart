import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/app/application/app_auto_shutdown_controller.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/core/common/api_server_state.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:gopeed/core/common/task_event.dart';

void main() {
  late StreamController<TaskEvent> events;

  setUp(() {
    events = StreamController<TaskEvent>.broadcast();
  });

  tearDown(() async {
    await events.close();
  });

  Future<_TestAutoShutdownController> start({
    required bool enabled,
    List<TaskActivity> activities = const [],
  }) async {
    final controller = _TestAutoShutdownController(events.stream, activities);
    final container = ProviderContainer(
      overrides: [
        appRuntimeControllerProvider.overrideWith(() => _TestRuntimeController(enabled)),
        appAutoShutdownControllerProvider.overrideWith(() => controller),
      ],
    );
    addTearDown(container.dispose);
    await container.read(appRuntimeControllerProvider.future);
    await container.read(appAutoShutdownControllerProvider.future);
    await container.pump();
    return controller;
  }

  Future<void> settle() => Future<void>.delayed(const Duration(milliseconds: 50));

  test('stays out of the way while the switch is off', () async {
    final controller = await start(enabled: false);

    events.add(const TaskEvent(type: TaskEventType.done, taskId: '1', name: 'archive.zip'));
    await settle();

    expect(controller.samples, 0);
    expect(controller.exitCount, 0);
  });

  test('exits after the last task finishes', () async {
    final controller = await start(enabled: true);

    events.add(const TaskEvent(type: TaskEventType.done, taskId: '1', name: 'archive.zip'));
    await settle();

    expect(controller.samples, 1);
    expect(controller.exitCount, 1);
  });

  test('a failing last task exits as well', () async {
    final controller = await start(enabled: true);

    events.add(const TaskEvent(type: TaskEventType.error, taskId: '1', name: 'archive.zip', error: '404'));
    await settle();

    expect(controller.exitCount, 1);
  });

  test('keeps the app open while another task is still downloading', () async {
    final controller = await start(
      enabled: true,
      activities: const [TaskActivity(activeDownloads: 1, pendingExtractions: 0)],
    );

    events.add(const TaskEvent(type: TaskEventType.done, taskId: '1', name: 'archive.zip'));
    await settle();

    expect(controller.samples, 1);
    expect(controller.exitCount, 0);
  });

  test('waits for the extraction started by the finished download', () async {
    final controller = await start(
      enabled: true,
      activities: const [
        TaskActivity(activeDownloads: 0, pendingExtractions: 1),
        TaskActivity(activeDownloads: 0, pendingExtractions: 0),
      ],
    );

    events.add(const TaskEvent(type: TaskEventType.done, taskId: '1', name: 'archive.rar'));
    await settle();

    expect(controller.samples, 2);
    expect(controller.exitCount, 1);
  });
}

class _TestAutoShutdownController extends AppAutoShutdownController {
  _TestAutoShutdownController(this.taskEvents, this.activities);

  @override
  final Stream<TaskEvent> taskEvents;

  /// Sampled results handed out in order; an exhausted list reads as idle.
  final List<TaskActivity> activities;

  int samples = 0;
  int exitCount = 0;

  @override
  Duration get settleDelay => Duration.zero;

  @override
  Duration get pollInterval => Duration.zero;

  @override
  Future<TaskActivity> readActivity() async {
    final index = samples++;
    return index < activities.length
        ? activities[index]
        : const TaskActivity(activeDownloads: 0, pendingExtractions: 0);
  }

  @override
  Future<void> shutdown() async {
    exitCount++;
  }
}

class _TestRuntimeController extends AppRuntimeController {
  _TestRuntimeController(this.enabled);

  final bool enabled;

  @override
  Future<AppRuntimeState> build() async => AppRuntimeState(
    startConfig: StartConfig(),
    apiServerState: ApiServerState.fromJson({}),
    downloaderConfig: DownloaderConfig()..extra.autoCloseWhenAllTasksDone = enabled,
  );
}
