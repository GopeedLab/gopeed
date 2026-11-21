import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/api/model/meta.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/task.dart' as api_task;
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
    return controller;
  }

  Future<void> settle() => Future<void>.delayed(const Duration(milliseconds: 50));

  group('shutdown trigger', () {
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

    test('keeps the app open when the last task fails', () async {
      final controller = await start(enabled: true);

      events.add(const TaskEvent(type: TaskEventType.error, taskId: '1', name: 'archive.zip', error: '404'));
      await settle();

      expect(controller.samples, 1);
      expect(controller.exitCount, 0);
    });

    test('a later successful task still exits after a failed one', () async {
      final controller = await start(enabled: true);

      events.add(const TaskEvent(type: TaskEventType.error, taskId: '1', name: 'a.zip', error: '404'));
      await settle();
      events.add(const TaskEvent(type: TaskEventType.done, taskId: '2', name: 'b.zip'));
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
  });

  group('TaskActivity.of', () {
    test('counts a downloading task as active', () {
      final activity = TaskActivity.of([_task(status: api_task.Status.running), _task(status: api_task.Status.wait)]);

      expect(activity.activeDownloads, 2);
    });

    test('ignores a paused task, it never finishes on its own', () {
      final activity = TaskActivity.of([_task(status: api_task.Status.pause)]);

      expect(activity.activeDownloads, 0);
    });

    test('counts a seeding torrent as active, the backend keeps it at done', () {
      final activity = TaskActivity.of([_task(status: api_task.Status.done, uploading: true)]);

      expect(activity.activeDownloads, 1);
    });

    test('ignores a paused torrent that was seeding', () {
      final activity = TaskActivity.of([_task(status: api_task.Status.pause, uploading: true)]);

      expect(activity.activeDownloads, 0);
    });

    test('ignores a finished task that is only kept around for its file', () {
      final activity = TaskActivity.of([_task(status: api_task.Status.done)]);

      expect(activity.activeDownloads, 0);
    });

    test('blocks the exit while an archive is queued, extracting or waiting for parts', () {
      final activity = TaskActivity.of([
        _task(extractStatus: api_task.ExtractStatus.queued),
        _task(extractStatus: api_task.ExtractStatus.extracting),
        _task(extractStatus: api_task.ExtractStatus.waitingParts),
      ]);

      expect(activity.pendingExtractions, 3);
    });

    test('lets the exit go once the extraction finished or failed', () {
      final activity = TaskActivity.of([
        _task(extractStatus: api_task.ExtractStatus.done),
        _task(extractStatus: api_task.ExtractStatus.error),
        _task(),
      ]);

      expect(activity.pendingExtractions, 0);
    });
  });
}

api_task.Task _task({
  api_task.Status status = api_task.Status.done,
  bool uploading = false,
  api_task.ExtractStatus extractStatus = api_task.ExtractStatus.none,
}) {
  final timestamp = DateTime(2026, 1, 1);
  return api_task.Task(
    id: 'task',
    name: 'file',
    meta: Meta(req: Request(url: 'https://example.com/file'), opts: Options()),
    status: status,
    uploading: uploading,
    progress: Progress(used: 0, speed: 0, downloaded: 0, uploadSpeed: 0, uploaded: 0, extractStatus: extractStatus),
    createdAt: timestamp,
    updatedAt: timestamp,
  );
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
