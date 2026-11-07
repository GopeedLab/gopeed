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

  setUp(() async {
    debugDefaultTargetPlatformOverride = TargetPlatform.windows;
    container = ProviderContainer(
      overrides: [
        appRuntimeControllerProvider.overrideWith(_Runtime.new),
        tasksControllerProvider.overrideWith(_Tasks.new),
        downloadProgressWindowControllerProvider.overrideWith(_Windows.new),
      ],
    );
    await container.read(appRuntimeControllerProvider.future);
    await container.read(tasksControllerProvider.future);
    container.read(downloadProgressWindowControllerProvider);
    windows = container.read(downloadProgressWindowControllerProvider.notifier) as _Windows;
    tasks = container.read(tasksControllerProvider.notifier) as _Tasks;
    runtime = container.read(appRuntimeControllerProvider.notifier) as _Runtime;
  });

  tearDown(() {
    container.dispose();
    debugDefaultTargetPlatformOverride = null;
  });

  test('ignores startup tasks and opens each new task once, including fast completions', () {
    expect(windows.opened, isEmpty);
    tasks.emit([_task('old'), _task('new'), _task('fast', status: TaskStatus.completed)]);
    expect(windows.opened, ['new', 'fast']);
    tasks.emit([_task('new', status: TaskStatus.paused)]);
    tasks.emit([_task('old'), _task('new')]);
    expect(windows.opened, ['new', 'fast']);
  });

  test('disabling applies immediately and enabling does not backfill old downloads', () {
    runtime.enable(false);
    tasks.emit([_task('old'), _task('disabled')]);
    expect(windows.opened, isEmpty);
    runtime.enable(true);
    tasks.emit([_task('old'), _task('disabled'), _task('enabled')]);
    expect(windows.opened, ['enabled']);
  });

  test('a group opens one window and refresh failures do not reset deduplication', () {
    tasks.emit([
      _task('old'),
      _task('group', members: [_task('member-1'), _task('member-2')]),
    ]);
    expect(windows.opened, ['group']);
    tasks.fail();
    tasks.emit([_task('old'), _task('group'), _task('member-1'), _task('next')]);
    expect(windows.opened, ['group', 'next']);
  });

  test('main app tasks never open automatically', () {
    tasks.emit([_task('manual', browserCaptured: false), _task('browser')]);
    expect(windows.opened, ['browser']);
  });

  test('mobile never opens automatic windows', () {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    container.invalidate(downloadProgressWindowControllerProvider);
    container.read(downloadProgressWindowControllerProvider);
    tasks.emit([_task('mobile')]);
    expect(windows.opened, isEmpty);
  });
}

class _Windows extends DownloadProgressWindowController {
  final opened = <String>[];
  @override
  Future<void> openWindow(String taskId) async => opened.add(taskId);
}

class _Tasks extends TasksController {
  @override
  Future<TasksState> build() async => TasksState(tasks: [_task('old')]);
  void emit(List<TaskRecord> tasks) => state = AsyncData(TasksState(tasks: tasks));
  void fail() => state = AsyncError(StateError('offline'), StackTrace.current);
}

class _Runtime extends AppRuntimeController {
  @override
  Future<AppRuntimeState> build() async => AppRuntimeState(
    startConfig: StartConfig(),
    apiServerState: ApiServerState.fromJson({}),
    downloaderConfig: DownloaderConfig()..extra.showDownloadProgressWindow = true,
  );

  void enable(bool enabled) => replaceDownloaderConfig(DownloaderConfig()..extra.showDownloadProgressWindow = enabled);
}

TaskRecord _task(
  String id, {
  TaskStatus status = TaskStatus.downloading,
  bool browserCaptured = true,
  List<TaskRecord> members = const [],
}) => TaskRecord(
  id: id,
  browserCaptured: browserCaptured,
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
