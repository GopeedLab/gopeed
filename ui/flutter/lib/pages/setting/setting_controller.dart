import 'package:get/get.dart';

class SettingController extends GetxController {
  final basicTapStatues = <int, bool>{}.obs;
  final advancedTapStatues = <int, bool>{}.obs;

  void clearTapStatus() {
    basicTapStatues.clear();
    advancedTapStatues.clear();
  }
}
