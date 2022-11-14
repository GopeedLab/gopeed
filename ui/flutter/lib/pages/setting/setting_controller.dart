import 'package:get/get.dart';

class SettingController extends GetxController {
  final tapStatues = <bool>[].obs;

  void clearTapStatus() {
    // set all tap status to false
    tapStatues.value = tapStatues.map((e) => false).toList();
  }
}
