import 'dart:async';

import 'package:get/get.dart';

import '../../../../api/api.dart';
import '../../../../api/model/task.dart';

abstract class TaskListController extends GetxController {
  List<Status> statuses;
  int Function(Task a, Task b) compare;

  TaskListController(this.statuses, this.compare);

  final tasks = <Task>[].obs;
  final selectedTaskIds = <String>[].obs;
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
    tasks.sort(compare);
    this.tasks.value = tasks;
  }
}
