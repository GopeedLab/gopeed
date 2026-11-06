import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../domain/download_speed_history.dart';
import '../domain/task_record.dart';
import 'tasks_controller.dart';

final downloadSpeedHistoryProvider = StreamProvider.autoDispose.family<DownloadSpeedHistory, String>((ref, id) {
  final stream = StreamController<DownloadSpeedHistory>();
  var history = const DownloadSpeedHistory();
  // History belongs to this window session; never backfill unobserved seconds.
  ref.listen(tasksControllerProvider, (_, next) {
    if (next.isLoading || next.hasError || next.value?.lastError != null) return;
    final task = next.value?.findTask(id);
    if (task == null) return;
    if (task.status == TaskStatus.completed) {
      if (history.samples.isEmpty) stream.add(history);
      return;
    }
    final speed = task.status == TaskStatus.downloading && !task.waiting ? task.speedBytes : 0;
    if (speed == null) return;
    history = history.record(DateTime.now(), speed);
    stream.add(history);
  }, fireImmediately: true);
  ref.onDispose(() => unawaited(stream.close()));
  return stream.stream;
});
