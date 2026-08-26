import 'package:get/get.dart';

import '../controllers/create_controller.dart';

class CreateBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<CreateController>(
      () => CreateController(),
    );
  }
}
