import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/widget/buid_task_list_view.dart';

import 'downloaded_controller.dart';

class DownloadedView extends GetView<DownloadedController> {
  const DownloadedView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return BuildTaskListView(tasks: controller.tasks);
  }
}
