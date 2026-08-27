import 'dart:async';

import 'package:get/get.dart';

import '../../../../api/api.dart';
import '../../../../api/model/task.dart';
import 'task_controller.dart';

abstract class TaskListController extends GetxController {
  List<Status> statuses;
  int Function(Task a, Task b) compare;

  TaskListController(this.statuses, this.compare);

  final tasks = <Task>[].obs;
  final selectedTaskIds = <String>[].obs;
  final isRunning = false.obs;

  /// Bounded rolling speed history buffer (max 60 samples per task)
  static const int maxSpeedSamples = 60;
  final speedHistory = <String, List<double>>{}.obs;

  late final Timer _timer;

  @override
  void onInit() async {
    super.onInit();

    start();
    _timer = Timer.periodic(const Duration(milliseconds: 1000), (timer) async {
      if (isRunning.value) {
        await getTasksState();
      }
    });
  }

  @override
  void onClose() {
    super.onClose();
    _timer.cancel();
  }

  void start() async {
    await getTasksState();
    isRunning.value = true;
  }

  void stop() {
    isRunning.value = false;
  }

  getTasksState() async {
    final tasks = await getTasks(statuses);
    // sort tasks by create time
    tasks.sort(compare);
    updateSpeedHistory(tasks);
    this.tasks.value = tasks;

    // Keep selected task synchronized with live state in Task Detail drawer
    if (Get.isRegistered<TaskController>()) {
      final taskController = Get.find<TaskController>();
      final selectedId = taskController.selectTask.value?.id;
      if (selectedId != null) {
        final updatedTask = tasks.firstWhereOrNull((t) => t.id == selectedId);
        if (updatedTask != null) {
          taskController.selectTask.value = updatedTask;
        }
      }
    }
  }

  /// Updates rolling speed samples for active tasks and cleans up deleted tasks.
  /// Stops appending once a task reaches a terminal state (Done or Error).
  void updateSpeedHistory(List<Task> currentTasks) {
    final currentTaskIds = currentTasks.map((t) => t.id).toSet();

    // Clear buffer for tasks that were deleted / no longer exist
    speedHistory.removeWhere((id, _) => !currentTaskIds.contains(id));

    // Update speed history for each task
    for (final task in currentTasks) {
      // Stop appending once a task reaches a terminal state (Done or Error)
      if (task.status == Status.done || task.status == Status.error) {
        continue;
      }

      final list = speedHistory.putIfAbsent(task.id, () => <double>[]);
      list.add(task.progress.speed.toDouble());
      if (list.length > maxSpeedSamples) {
        list.removeAt(0);
      }
    }
    speedHistory.refresh();
  }

  /// Gets the rolling speed history buffer for a given task ID.
  List<double> getSpeedHistory(String taskId) {
    return speedHistory[taskId] ?? [];
  }

  /// Clears the speed history buffer for a specific task or all tasks.
  void clearSpeedHistory([String? taskId]) {
    if (taskId != null) {
      speedHistory.remove(taskId);
    } else {
      speedHistory.clear();
    }
    speedHistory.refresh();
  }
}
