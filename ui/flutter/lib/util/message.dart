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

void showMessage(title, msg) {
  Get.snackbar(title, msg);
}
