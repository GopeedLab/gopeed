import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../controllers/redirect_controller.dart';

class RedirectArgs {
  final String page;
  final dynamic arguments;

  RedirectArgs(this.page, {this.arguments});
}

class RedirectView extends GetView<RedirectController> {
  const RedirectView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final redirectArgs = Get.rootDelegate.arguments() as RedirectArgs;
    // Waiting for previous page controller to delete, avoid deleting controller that route page after redirect
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      await Future.delayed(const Duration(milliseconds: 350));
      Get.rootDelegate
          .offAndToNamed(redirectArgs.page, arguments: redirectArgs.arguments);
    });
    return const SizedBox();
  }
}
