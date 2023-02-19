import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../views/views/buid_task_list_view.dart';
import '../controllers/downloading_controller.dart';

class DownloadingView extends GetView<DownloadingController> {
  const DownloadingView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return BuildTaskListView(tasks: controller.tasks);
  }
}
