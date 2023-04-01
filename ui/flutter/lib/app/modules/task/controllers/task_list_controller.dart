import 'dart:async';

import 'package:get/get.dart';

import '../../../../api/api.dart';
import '../../../../api/model/task.dart';

abstract class TaskListController extends GetxController {
  List<Status> statuses;

  TaskListController(this.statuses);

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
    tasks.value = await getTasks(statuses);
  }
}
