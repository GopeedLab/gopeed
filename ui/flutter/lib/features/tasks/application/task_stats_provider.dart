import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/model/task.dart';
import '../../../api/model/task_stats.dart';
import '../../../core/capabilities/app_capabilities.dart';

typedef TaskStatsRequest = ({String taskId, Protocol? protocol});

final taskStatsProvider = FutureProvider.autoDispose.family<TaskStats?, TaskStatsRequest>((ref, request) async {
  final refreshTimer = Timer(const Duration(seconds: 2), ref.invalidateSelf);
  ref.onDispose(refreshTimer.cancel);
  final json = await ref.read(gopeedServiceProvider).getTaskStats(request.taskId);
  return TaskStats.fromJson(request.protocol, json);
});
