import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/core/common/api_server_state.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:gopeed/features/tasks/application/download_progress_window_controller.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';

void main() {
  late ProviderContainer container;
  late _Windows windows;
  late _Tasks tasks;
  late _Runtime runtime;
  late Completer<TasksState> firstSnapshot;
  final old = _task('old', captureSession: 'previous-session');

  TaskRecord captured(String id, {TaskStatus status = TaskStatus.downloading, List<TaskRecord> members = const []}) =>
      _task(id, captureSession: windows.captureSession, status: status, members: members);

  Future<void> finishStartup([List<TaskRecord>? initialTasks]) async {
    firstSnapshot.complete(TasksState(tasks: initialTasks ?? [old]));
    await container.read(tasksControllerProvider.future);
  }

  setUp(() async {
    debugDefaultTargetPlatformOverride = TargetPlatform.windows;
    firstSnapshot = Completer<TasksState>();
    container = ProviderContainer(
      overrides: [
        appRuntimeControllerProvider.overrideWith(_Runtime.new),
        tasksControllerProvider.overrideWith(() => _Tasks(firstSnapshot.future)),
        downloadProgressWindowControllerProvider.overrideWith(_Windows.new),
      ],
    );
    await container.read(appRuntimeControllerProvider.future);
    windows = container.read(downloadProgressWindowControllerProvider.notifier) as _Windows;
    tasks = container.read(tasksControllerProvider.notifier) as _Tasks;
    runtime = container.read(appRuntimeControllerProvider.notifier) as _Runtime;
  });

  tearDown(() {
    container.dispose();
    debugDefaultTargetPlatformOverride = null;
  });

  test('opens a cold-start capture in the first snapshot only once', () async {
    final initialTasks = [old, _task('legacy', captureSession: 'true'), _task('manual'), captured('cold')];
    await finishStartup(initialTasks);
    expect(windows.opened, ['cold']);
    tasks.emit(initialTasks);
    expect(windows.opened, ['cold']);
  });

  test('ignores previous captures and opens each new task once, including fast completions', () async {
    await finishStartup();
    expect(windows.opened, isEmpty);
    tasks.emit([old, captured('new'), captured('fast', status: TaskStatus.completed)]);
    expect(windows.opened, ['new', 'fast']);
    tasks.emit([captured('new', status: TaskStatus.paused)]);
    tasks.emit([old, captured('new')]);
    expect(windows.opened, ['new', 'fast']);
  });

  test('the saved off setting applies from startup and enabling does not backfill captures', () async {
    runtime.enable(false);
    await finishStartup([old, captured('disabled')]);
    expect(windows.opened, isEmpty);
    runtime.enable(true);
    tasks.emit([old, captured('disabled'), captured('enabled')]);
    expect(windows.opened, ['enabled']);
  });

  test('a group opens one window and refresh failures do not reset deduplication', () async {
    await finishStartup();
    tasks.emit([
      old,
      captured('group', members: [captured('member-1'), captured('member-2')]),
    ]);
    expect(windows.opened, ['group']);
    tasks.fail();
    tasks.emit([old, captured('group'), captured('member-1'), captured('next')]);
    expect(windows.opened, ['group', 'next']);
  });

  test('main app tasks never open automatically', () async {
    await finishStartup();
    tasks.emit([_task('manual'), captured('browser')]);
    expect(windows.opened, ['browser']);
  });

  test('mobile never opens automatic windows', () async {
    await finishStartup();
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    container.invalidate(downloadProgressWindowControllerProvider);
    windows = container.read(downloadProgressWindowControllerProvider.notifier) as _Windows;
    tasks.emit([captured('mobile')]);
    expect(windows.opened, isEmpty);
  });
}

class _Windows extends DownloadProgressWindowController {
  final opened = <String>[];
  @override
  Future<void> openWindow(String taskId) async => opened.add(taskId);
}

class _Tasks extends TasksController {
  _Tasks(this.initial);

  final Future<TasksState> initial;

  @override
  Future<TasksState> build() => initial;
  void emit(List<TaskRecord> tasks) => state = AsyncData(TasksState(tasks: tasks));
  void fail() => state = AsyncError(StateError('offline'), StackTrace.current);
}

class _Runtime extends AppRuntimeController {
  @override
  Future<AppRuntimeState> build() async => AppRuntimeState(
    startConfig: StartConfig(),
    apiServerState: ApiServerState.fromJson({}),
    downloaderConfig: DownloaderConfig(),
  );

  void enable(bool enabled) => replaceDownloaderConfig(DownloaderConfig()..extra.showDownloadProgressWindow = enabled);
}

TaskRecord _task(
  String id, {
  TaskStatus status = TaskStatus.downloading,
  String? captureSession,
  List<TaskRecord> members = const [],
}) => TaskRecord(
  id: id,
  browserCaptureSession: captureSession,
  name: id,
  status: status,
  downloaded: '0 B',
  url: 'https://example.com/$id',
  storagePath: '/downloads/$id',
  files: const [],
  uploading: false,
  isGroup: members.isNotEmpty,
  members: members,
);
