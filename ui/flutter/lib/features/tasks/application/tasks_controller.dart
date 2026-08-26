import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/model/request.dart';
import '../../../api/model/resolve_task.dart';
import '../../../api/model/task.dart' as api_model;
import '../../../core/capabilities/app_capabilities.dart';
import '../domain/task_record.dart';

final tasksControllerProvider = AsyncNotifierProvider<TasksController, TasksState>(TasksController.new);

class TasksState {
  const TasksState({
    required this.tasks,
    this.downloadSpeedBytesPerSecond = 0,
    this.uploadSpeedBytesPerSecond = 0,
    this.lastError,
  });

  final List<TaskRecord> tasks;
  final int downloadSpeedBytesPerSecond;
  final int uploadSpeedBytesPerSecond;
  final Object? lastError;

  TasksState copyWith({
    List<TaskRecord>? tasks,
    int? downloadSpeedBytesPerSecond,
    int? uploadSpeedBytesPerSecond,
    Object? lastError,
  }) {
    return TasksState(
      tasks: tasks ?? this.tasks,
      downloadSpeedBytesPerSecond: downloadSpeedBytesPerSecond ?? this.downloadSpeedBytesPerSecond,
      uploadSpeedBytesPerSecond: uploadSpeedBytesPerSecond ?? this.uploadSpeedBytesPerSecond,
      lastError: lastError,
    );
  }
}

({int downloadBytesPerSecond, int uploadBytesPerSecond}) aggregateTaskTransferSpeeds(Iterable<api_model.Task> tasks) {
  var downloadBytesPerSecond = 0;
  var uploadBytesPerSecond = 0;
  for (final task in tasks) {
    if (task.status == api_model.Status.running) {
      downloadBytesPerSecond += task.progress.speed > 0 ? task.progress.speed : 0;
    }
    if (task.uploading && (task.status == api_model.Status.running || task.status == api_model.Status.done)) {
      uploadBytesPerSecond += task.progress.uploadSpeed > 0 ? task.progress.uploadSpeed : 0;
    }
  }
  return (downloadBytesPerSecond: downloadBytesPerSecond, uploadBytesPerSecond: uploadBytesPerSecond);
}

class TasksController extends AsyncNotifier<TasksState> {
  Timer? _timer;

  @override
  Future<TasksState> build() async {
    _timer = Timer.periodic(const Duration(seconds: 1), (_) => unawaited(refresh(silent: true)));
    ref.onDispose(() => _timer?.cancel());
    return _fetch();
  }

  Future<void> refresh({bool silent = false}) async {
    if (!silent) {
      state = const AsyncValue.loading();
    }
    try {
      state = AsyncValue.data(await _fetch());
    } catch (error, stackTrace) {
      final previous = state.value;
      if (previous != null && silent) {
        state = AsyncValue.data(previous.copyWith(lastError: error));
      } else {
        state = AsyncValue.error(error, stackTrace);
      }
    }
  }

  Future<void> pause(String id) async {
    await ref.read(gopeedServiceProvider).pauseTask(id);
    await refresh(silent: true);
  }

  Future<void> resume(String id) async {
    await ref.read(gopeedServiceProvider).continueTask(id);
    await refresh(silent: true);
  }

  Future<void> updateUrl(String id, String url, {Map<String, String> headers = const {}}) async {
    await ref
        .read(gopeedServiceProvider)
        .patchTask(
          id,
          ResolveTask(
            req: Request(
              url: url,
              extra: ReqExtraHttp(header: headers).toJson(),
            ),
          ),
        );
    await ref.read(gopeedServiceProvider).continueTask(id);
    await refresh(silent: true);
  }

  Future<void> pauseAll(List<String>? ids) async {
    await ref.read(gopeedServiceProvider).pauseAllTasks(ids);
    await refresh(silent: true);
  }

  Future<void> resumeAll(List<String>? ids) async {
    await ref.read(gopeedServiceProvider).continueAllTasks(ids);
    await refresh(silent: true);
  }

  Future<void> deleteSelected(List<String>? ids, {bool force = false}) async {
    await ref.read(gopeedServiceProvider).deleteTasks(ids, force);
    await refresh(silent: true);
  }

  Future<TasksState> _fetch() async {
    final tasks = await ref.read(gopeedServiceProvider).getTasks(api_model.Status.values);
    final transferSpeeds = aggregateTaskTransferSpeeds(tasks);
    tasks.sort((a, b) {
      if (a.status == api_model.Status.running && b.status != api_model.Status.running) return -1;
      if (a.status != api_model.Status.running && b.status == api_model.Status.running) return 1;
      return b.updatedAt.compareTo(a.updatedAt);
    });
    return TasksState(
      tasks: tasks.map(TaskRecord.fromApi).toList(growable: false),
      downloadSpeedBytesPerSecond: transferSpeeds.downloadBytesPerSecond,
      uploadSpeedBytesPerSecond: transferSpeeds.uploadBytesPerSecond,
    );
  }
}
