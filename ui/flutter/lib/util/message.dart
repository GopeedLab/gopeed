import 'package:get/get.dart';

import '../api/model/result.dart';

void showErrorMessage(msg) {
  final title = 'error'.tr;
  if (msg is Result) {
    Get.snackbar(title, msg.msg!);
    return;
  }
  if (msg is Exception && (msg as dynamic).message is Result) {
    Get.snackbar(title, ((msg as dynamic).message as Result).msg!);
    return;
  }
  Get.snackbar(title, msg.toString());
}

var _showMessageFlag = true;

void showMessage(title, msg) {
  if (_showMessageFlag) {
    _showMessageFlag = false;
    Get.snackbar(title, msg);
    Future.delayed(const Duration(seconds: 3), () {
      _showMessageFlag = true;
    });
  }
}
