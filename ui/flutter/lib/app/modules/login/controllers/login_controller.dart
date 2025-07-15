import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../../api/api.dart' as api;
import '../../../../api/api.dart';
import '../../../../api/model/login.dart';
import '../../../../database/database.dart';
import '../../../../util/message.dart';
import '../../../routes/app_pages.dart';
import '../../app/controllers/app_controller.dart';

class LoginController extends GetxController {
  final formKey = GlobalKey<FormState>();
  final usernameController = TextEditingController();
  final passwordController = TextEditingController();

  final isLoading = false.obs;
  final passwordVisible = false.obs;

  @override
  void onClose() {
    usernameController.dispose();
    passwordController.dispose();
    super.onClose();
  }

  void togglePasswordVisibility() {
    passwordVisible.value = !passwordVisible.value;
  }

  Future<void> login() async {
    if (!formKey.currentState!.validate()) {
      return;
    }

    isLoading.value = true;
    try {
      final loginReq = LoginReq(
        username: usernameController.text.trim(),
        password: passwordController.text,
      );

      final token = await api.login(loginReq);
      // Login successful, save the token
      Database.instance.saveWebToken(token);
      // Reload config
      final controller = Get.put(AppController());
      await controller.loadDownloaderConfig();
      // Navigate to home page
      Get.rootDelegate.offAndToNamed(Routes.HOME);
    } catch (e) {
      if (e is TimeoutException) {
        showMessage('error'.tr, 'login_failed_network'.tr);
      } else {
        showMessage('error'.tr, 'login_failed'.tr);
      }
    } finally {
      isLoading.value = false;
    }
  }
}
