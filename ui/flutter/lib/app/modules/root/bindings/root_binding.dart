import 'package:get/get.dart';

import '../../../../util/network_monitor.dart';
import '../controllers/root_controller.dart';

class RootBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<RootController>(() => RootController(), fenix: true);
    Get.put<NetworkMonitor>(NetworkMonitor(), permanent: true);
  }
}
