import 'package:get/get.dart';
import 'package:gopeed/app/modules/lock/controllers/lock_setup_controller.dart';

class LockSetupBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<LockSetupController>(
      () => LockSetupController(),
    );
  }
}
