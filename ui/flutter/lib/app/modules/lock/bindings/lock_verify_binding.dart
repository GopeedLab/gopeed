import 'package:get/get.dart';
import 'package:gopeed/app/modules/lock/controllers/lock_verify_controller.dart';

class LockVerifyBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<LockVerifyController>(
      () => LockVerifyController(),
    );
  }
}
