import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/model/create_task.dart';

class PendingUpdateTask {
  const PendingUpdateTask({required this.id, required this.name});

  final String id;
  final String name;
}

final pendingUpdateTaskProvider = NotifierProvider<PendingUpdateTaskController, PendingUpdateTask?>(
  PendingUpdateTaskController.new,
);

class PendingUpdateTaskController extends Notifier<PendingUpdateTask?> {
  @override
  PendingUpdateTask? build() => null;

  void set(PendingUpdateTask? task) {
    state = task;
  }

  void clear() {
    state = null;
  }
}

class PendingUpdateRequest {
  const PendingUpdateRequest({required this.task, required this.createTask});

  final PendingUpdateTask task;
  final CreateTask createTask;
}

final pendingUpdateRequestProvider = NotifierProvider<PendingUpdateRequestController, PendingUpdateRequest?>(
  PendingUpdateRequestController.new,
);

class PendingUpdateRequestController extends Notifier<PendingUpdateRequest?> {
  @override
  PendingUpdateRequest? build() => null;

  void set(PendingUpdateRequest request) {
    state = request;
  }

  void clear() {
    state = null;
  }
}
