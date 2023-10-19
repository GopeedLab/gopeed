import 'package:get/get.dart';

import '../controllers/extension_controller.dart';

class ExtensionBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<ExtensionController>(
      () => ExtensionController(),
    );
  }
}
