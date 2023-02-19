import 'package:get/get.dart';

class SettingController extends GetxController {
  final tapStatues = <String, bool>{}.obs;

  // set all tap status to false
  void clearTap() {
    tapStatues.updateAll((key, value) => false);
  }

  // set one tap status to true
  void onTap(String key) {
    clearTap();
    tapStatues[key] = true;
  }
}
