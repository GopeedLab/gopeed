import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../views/buid_task_list_view.dart';
import '../controllers/task_downloaded_controller.dart';

class TaskDownloadedView extends GetView<TaskDownloadedController> {
  const TaskDownloadedView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return BuildTaskListView(
        tasks: controller.tasks, selectedTaskIds: controller.selectedTaskIds);
  }
}
