import 'dart:convert';

import 'package:get/get.dart';
import 'package:gopeed/api/api.dart';

class SettingController extends GetxController {
  final tapStatues = <String, bool>{}.obs;
  final latestVersion = "".obs;

  @override
  void onInit() {
    super.onInit();
    fetchLatestVersion();
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
    String? releaseDataStr;
    try {
      releaseDataStr =
          (await proxyRequest("https://gopeed.com/api/release")).data;
    } catch (e) {
      releaseDataStr = (await proxyRequest(
              "https://api.github.com/repos/GopeedLab/gopeed/releases/latest"))
          .data;
    }
    if (releaseDataStr == null) {
      return;
    }
    final releaseData = jsonDecode(releaseDataStr);
    latestVersion.value = releaseData["tag_name"].substring(1);
  }
}
