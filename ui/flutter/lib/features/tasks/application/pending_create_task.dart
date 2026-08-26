import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/model/create_task.dart';

final pendingCreateTaskProvider = NotifierProvider<PendingCreateTask, CreateTask?>(PendingCreateTask.new);

class PendingCreateTask extends Notifier<CreateTask?> {
  @override
  CreateTask? build() => null;

  void set(CreateTask? task) {
    state = task;
  }

  void clear() {
    state = null;
  }
}
