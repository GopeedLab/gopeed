import 'package:flutter_riverpod/flutter_riverpod.dart';

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
