import 'package:get/get.dart';

import '../controllers/downloading_controller.dart';

class DownloadingBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<DownloadingController>(
      () => DownloadingController(),
    );
  }
}
