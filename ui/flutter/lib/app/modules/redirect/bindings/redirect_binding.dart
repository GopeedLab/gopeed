import 'package:get/get.dart';

import '../controllers/redirect_controller.dart';

class RedirectBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<RedirectController>(
      () => RedirectController(),
    );
  }
}
