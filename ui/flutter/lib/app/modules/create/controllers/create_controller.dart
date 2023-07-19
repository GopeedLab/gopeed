import 'package:flutter/material.dart';
import 'package:get/get.dart';

class CreateController extends GetxController
    with GetSingleTickerProviderStateMixin {
  // final files = [].obs;
  final RxList fileInfos = [].obs;
  final RxList selectedIndexes = [].obs;
  final RxList openedFolders = [].obs;
  final isResolving = false.obs;
  final showAdvanced = false.obs;
  late TabController advancedTabController;

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
}
