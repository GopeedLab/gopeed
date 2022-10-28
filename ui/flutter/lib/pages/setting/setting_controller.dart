import 'package:get/get.dart';

import '../../setting/setting.dart';

class SettingController extends GetxController {
  final setting = Setting.instance.obs;
  final tapStatues = <int, bool>{}.obs;
}
