import 'package:get/get.dart';

import '../../../../util/network_monitor.dart';
import '../controllers/app_controller.dart';

class AppBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<AppController>(
      () => AppController(),
    );
    Get.put<NetworkMonitor>(NetworkMonitor(), permanent: true);
  }
}
