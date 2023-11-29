import 'dart:async';

import 'package:get/get.dart';

import '../../../../api/api.dart';
import '../../../../api/model/task.dart';

enum SortDirection { asc, desc }

abstract class TaskListController extends GetxController {
  List<Status> statuses;
  SortDirection sortDirection;

  TaskListController(this.statuses, this.sortDirection);

  final tasks = <Task>[].obs;
  final isRunning = false.obs;

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
    tasks.sort((a, b) {
      if (sortDirection == SortDirection.asc) {
        return a.createdAt.compareTo(b.createdAt);
      } else {
        return b.createdAt.compareTo(a.createdAt);
      }
    });
    this.tasks.value = tasks;
  }
}
