import 'package:get/get.dart';

class SettingController extends GetxController {
  final tapStatues = <int, bool>{}.obs;

  void clearTapStatus() {
    tapStatues.clear();
  }
}
