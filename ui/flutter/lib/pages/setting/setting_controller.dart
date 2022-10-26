import 'package:get/get.dart';
import 'package:gopeed/api/api.dart';
import 'package:gopeed/api/model/server_config.dart';

import '../../setting/setting.dart';

class SettingController extends GetxController {
  final setting = Setting.instance.obs;
  final tapStatues = <int, bool>{}.obs;
}
