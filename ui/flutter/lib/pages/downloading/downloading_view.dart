import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../widget/buid_task_list_view.dart';
import 'downloading_controller.dart';

class DownloadingView extends GetView<DownloadingController> {
  const DownloadingView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return BuildTaskListView(tasks: controller.tasks);
  }
}
