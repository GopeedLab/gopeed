import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/model/task.dart';
import '../../../core/capabilities/app_capabilities.dart';

final taskRuntimeStatusProvider = FutureProvider.autoDispose.family<TaskRuntimeStatus, String>((ref, taskId) async {
  final status = await ref.read(gopeedServiceProvider).getTaskStatus(taskId);
  final refreshTimer = Timer(const Duration(seconds: 1), ref.invalidateSelf);
  ref.onDispose(refreshTimer.cancel);
  return status;
});
