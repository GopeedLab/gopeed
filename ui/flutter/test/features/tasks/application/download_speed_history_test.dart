import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/application/download_speed_history_provider.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/tasks/domain/download_speed_history.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';

void main() {
  test('history retains the entire session, with gaps preserved and one sample per second', () {
    final start = DateTime.utc(2026);
    var history = const DownloadSpeedHistory();
    for (var second = 0; second <= 70; second++) {
      if (second >= 30 && second < 40) continue;
      history = history.record(start.add(Duration(seconds: second)), second * 100);
    }
    expect(history.samples.first.time, start);
    expect(history.samples.length, 61);
    expect(history.peak, 7000);
    history = history.record(start.add(const Duration(seconds: 70, milliseconds: 500)), -1);
    expect(history.samples.last.bytesPerSecond, 0);
    expect(history.samples.length, 61);
    expect(history.elapsed, const Duration(seconds: 70, milliseconds: 500));
  });

  test('time axis fills the width and compresses earlier samples as history grows', () {
    final start = DateTime.utc(2026);
    var history = const DownloadSpeedHistory().record(start, 100);
    expect(history.position(start), 0);
    history = history.record(start.add(const Duration(seconds: 10)), 200);
    expect(history.position(start), 0);
    expect(history.position(history.samples.last.time), 1);
    history = history.record(start.add(const Duration(seconds: 100)), 300);
    expect(history.position(start), 0);
    expect(history.position(start.add(const Duration(seconds: 10))), .1);
    expect(history.position(history.samples.last.time), 1);
  });

  test('average uses downloaded bytes and active duration, not chart samples', () {
    expect(_task().averageSpeedBytes, 512);
    expect(_task(duration: Duration.zero).averageSpeedBytes, isNull);
    expect(_task(duration: null).averageSpeedBytes, isNull);
    expect(_task(downloaded: null).averageSpeedBytes, isNull);
    expect(_task(duration: const Duration(milliseconds: 500)).averageSpeedBytes, 4096);
  });

  test('history observes task snapshots without recording stale API failures', () async {
    final container = ProviderContainer(overrides: [tasksControllerProvider.overrideWith(_Tasks.new)]);
    addTearDown(container.dispose);
    final provider = downloadSpeedHistoryProvider('one');
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    final history = await container.read(provider.future);
    expect(history.samples.single.bytesPerSecond, 1024);
    final controller = container.read(tasksControllerProvider.notifier) as _Tasks;
    controller.fail();
    await Future<void>.delayed(Duration.zero);
    expect(identical(container.read(provider).value, history), isTrue);
    controller.emitPaused();
    await Future<void>.delayed(Duration.zero);
    expect(container.read(provider).requireValue.samples.last.bytesPerSecond, 0);
  });

  test('completion freezes the curve despite further task polling', () async {
    final container = ProviderContainer(overrides: [tasksControllerProvider.overrideWith(_Tasks.new)]);
    addTearDown(container.dispose);
    final provider = downloadSpeedHistoryProvider('one');
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    final history = await container.read(provider.future);
    final controller = container.read(tasksControllerProvider.notifier) as _Tasks;
    for (var poll = 0; poll < 3; poll++) {
      controller.emitCompleted();
      await Future<void>.delayed(Duration.zero);
      expect(identical(container.read(provider).value, history), isTrue);
    }
    expect(history.samples.last.bytesPerSecond, 1024);
  });
}

class _Tasks extends TasksController {
  @override
  Future<TasksState> build() async => TasksState(tasks: [_task()]);
  void fail() => state = AsyncData(state.requireValue.copyWith(lastError: StateError('offline')));
  void emitPaused() => state = AsyncData(TasksState(tasks: [_task(status: TaskStatus.paused)]));
  void emitCompleted() => state = AsyncData(TasksState(tasks: [_task(status: TaskStatus.completed)]));
}

TaskRecord _task({
  Duration? duration = const Duration(seconds: 4),
  int? downloaded = 2048,
  TaskStatus status = TaskStatus.downloading,
}) => TaskRecord(
  id: 'one',
  name: 'one.zip',
  status: status,
  downloaded: '2 KB',
  total: '4 KB',
  downloadedBytes: downloaded,
  downloadDuration: duration,
  speedBytes: 1024,
  url: 'https://example.com/one.zip',
  storagePath: '/one.zip',
  files: const [],
  uploading: false,
);
