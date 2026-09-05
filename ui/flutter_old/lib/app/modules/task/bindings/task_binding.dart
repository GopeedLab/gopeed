import 'package:get/get.dart';

import '../controllers/task_controller.dart';
import '../controllers/task_downloaded_controller.dart';
import '../controllers/task_downloading_controller.dart';

class TaskBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<TaskController>(
      () => TaskController(),
    );
    Get.lazyPut<TaskDownloadingController>(
      () => TaskDownloadingController(),
    );
    Get.lazyPut<TaskDownloadedController>(
      () => TaskDownloadedController(),
    );
  }
}
