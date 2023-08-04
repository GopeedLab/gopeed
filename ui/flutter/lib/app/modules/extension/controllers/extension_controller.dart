import 'package:get/get.dart';
import 'package:gopeed/api/api.dart';

import '../../../../api/model/extension.dart';

class ExtensionController extends GetxController {
  var extensions = <Extension>[].obs;

  @override
  void onInit() {
    super.onInit();
    load();
  }

  void load() async {
    extensions.value = await getExtensions();
  }
}
