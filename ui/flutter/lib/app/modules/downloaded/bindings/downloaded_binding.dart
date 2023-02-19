import 'package:get/get.dart';

import '../controllers/downloaded_controller.dart';

class DownloadedBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<DownloadedController>(
      () => DownloadedController(),
    );
  }
}
