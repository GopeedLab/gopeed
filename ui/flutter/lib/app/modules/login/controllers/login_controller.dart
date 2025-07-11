import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../../api/api.dart' as api;
import '../../../../api/model/login.dart';
import '../../../../util/message.dart';
import '../../../routes/app_pages.dart';

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

      await api.login(loginReq);

      // 登录成功，跳转到主页
      Get.rootDelegate.offAndToNamed(Routes.HOME);
      showMessage('success'.tr, 'login_success'.tr);
    } catch (e) {
      showErrorMessage(e);
    } finally {
      isLoading.value = false;
    }
  }
}
