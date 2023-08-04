import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:get/get.dart';

class CreateController extends GetxController
    with GetSingleTickerProviderStateMixin {
  // final files = [].obs;
  final RxList fileInfos = [].obs;
  final RxList openedFolders = [].obs;
  final selectedIndexes = <int>[].obs;
  final isResolving = false.obs;
  final showAdvanced = false.obs;
  late TabController advancedTabController;
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
