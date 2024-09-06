import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/api/model/request.dart';

class CreateController extends GetxController
    with GetSingleTickerProviderStateMixin {
  // final files = [].obs;
  final RxList fileInfos = [].obs;
  final RxList openedFolders = [].obs;
  final selectedIndexes = <int>[].obs;
  final isConfirming = false.obs;
  final showAdvanced = false.obs;
  final directDownload = false.obs;
  final proxyConfig = Rx<RequestProxy?>(null);
  late TabController advancedTabController;
  final oldUrl = "".obs;
  final fileDataUri = "".obs;

  @override
  void onInit() {
    super.onInit();
    advancedTabController = TabController(length: 2, vsync: this);
  }

  @override
  void onClose() {
    advancedTabController.dispose();
    super.onClose();
  }

  void setFileDataUri(Uint8List bytes) {
    fileDataUri.value =
        "data:application/x-bittorrent;base64,${base64.encode(bytes)}";
  }

  void clearFileDataUri() {
    fileDataUri.value = "";
  }
}
