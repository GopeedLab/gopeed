import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';
import 'package:gopeed/api/model/task.dart';

class TaskController extends GetxController {
  final tabIndex = 0.obs;
  final scaffoldKey = GlobalKey<ScaffoldState>();
  final selectTask = Rx<Task?>(null);

  @override
  void onInit() {
    super.onInit();
    if (kIsWeb) {
      BrowserContextMenu.disableContextMenu();
    }
  }

  @override
  void onClose() {
    super.onClose();
    if (kIsWeb) {
      BrowserContextMenu.enableContextMenu();
    }
  }
}
