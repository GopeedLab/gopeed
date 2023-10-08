import 'package:get/get.dart';
import 'package:gopeed/api/api.dart';

import '../../../../api/model/extension.dart';

class ExtensionController extends GetxController {
  final extensions = <Extension>[].obs;
  final updateFlags = <String, String>{}.obs;
  final devMode = false.obs;
  var _devModeCount = 0;

  @override
  void onInit() async {
    super.onInit();
    await load();
    checkUpdate();
  }

  Future<void> load() async {
    extensions.value = await getExtensions();
  }

  Future<void> checkUpdate() async {
    for (final ext in extensions) {
      final resp = await upgradeCheckExtension(ext.identity);
      if (resp.newVersion.isNotEmpty) {
        updateFlags[ext.identity] = resp.newVersion;
      }
    }
  }

  // 在两秒内5次安装按钮进入开发者模式
  void tryOpenDevMode() {
    if (_devModeCount == 0) {
      Future.delayed(const Duration(seconds: 2), () {
        if (devMode.value) return;
        devMode.value = false;
        _devModeCount = 0;
      });
    }
    _devModeCount++;
    if (_devModeCount >= 5) {
      devMode.value = true;
    }
  }
}
