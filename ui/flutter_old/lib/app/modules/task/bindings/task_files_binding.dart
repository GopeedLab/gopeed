import 'package:get/get.dart';

import '../controllers/task_files_controller.dart';

class TaskFilesBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<TaskFilesController>(
      () => TaskFilesController(),
    );
  }
}
