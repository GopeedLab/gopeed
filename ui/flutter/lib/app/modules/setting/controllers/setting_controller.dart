import 'package:get/get.dart';

import '../../../../database/database.dart';
import '../../../../util/updater.dart';

class SettingController extends GetxController {
  final tapStatues = <String, bool>{}.obs;
  final latestVersion = Rxn<VersionInfo>();
  final networkAutoControl = false.obs;

  @override
  void onInit() {
    super.onInit();
    fetchLatestVersion();
    // Initialize network auto control setting
    networkAutoControl.value = Database.instance.getNetworkAutoControl();
  }

  // set all tap status to false
  void clearTap() {
    tapStatues.updateAll((key, value) => false);
  }

  // set one tap status to true
  void onTap(String key) {
    clearTap();
    tapStatues[key] = true;
  }

  // fetch latest version
  void fetchLatestVersion() async {
    latestVersion.value = await checkUpdate();
  }

  // update network auto control setting
  void updateNetworkAutoControl(bool value) {
    networkAutoControl.value = value;
    Database.instance.saveNetworkAutoControl(value);
  }
}
