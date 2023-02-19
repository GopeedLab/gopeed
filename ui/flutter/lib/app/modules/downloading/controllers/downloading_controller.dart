import 'dart:async';

import 'package:get/get.dart';

import '../../../../api/api.dart';
import '../../../../api/model/task.dart';

class DownloadingController extends GetxController {
  final tasks = <Task>[].obs;

  late final Timer _timer;

  @override
  void onInit() async {
    super.onInit();
    await getTasksState();
    _timer = Timer.periodic(const Duration(milliseconds: 1000), (timer) async {
      // debugPrint('timer${DateTime.now()}');
      await getTasksState();
    });
  }

  @override
  void onClose() {
    super.onClose();
    _timer.cancel();
  }

  getTasksState() async {
    tasks.value = await getTasks(
        [Status.ready, Status.running, Status.pause, Status.error]);
  }
}
