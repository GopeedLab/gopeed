import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../views/buid_task_list_view.dart';
import '../controllers/task_downloading_controller.dart';

class TaskDownloadingView extends GetView<TaskDownloadingController> {
  const TaskDownloadingView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return BuildTaskListView(
        tasks: controller.tasks, selectedTaskIds: controller.selectedTaskIds);
  }
}
