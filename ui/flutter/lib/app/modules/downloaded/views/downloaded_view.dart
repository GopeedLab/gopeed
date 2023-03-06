import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../views/views/buid_task_list_view.dart';
import '../controllers/downloaded_controller.dart';

class DownloadedView extends GetView<DownloadedController> {
  const DownloadedView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return BuildTaskListView(tasks: controller.tasks);
  }
}
