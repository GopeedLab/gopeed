import 'package:get/get.dart';
import 'package:gopeed/api/api.dart';

import '../../../../api/model/extension.dart';

class ExtensionController extends GetxController {
  final extensions = <Extension>[].obs;
  final updateFlags = <String, String>{}.obs;

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
}
