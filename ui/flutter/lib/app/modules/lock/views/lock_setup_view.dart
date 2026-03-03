import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:pinput/pinput.dart';
import 'package:gopeed/app/modules/lock/controllers/lock_setup_controller.dart';

const _gopeedGreen = Color(0xFF79C476);

class LockSetupView extends GetView<LockSetupController> {
  const LockSetupView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    final defaultPinTheme = PinTheme(
      width: 56,
      height: 56,
      textStyle: TextStyle(
        fontSize: 22,
        fontWeight: FontWeight.w600,
        color: isDark ? Colors.white : Colors.black87,
      ),
      decoration: BoxDecoration(
        color: isDark ? Colors.grey[850] : Colors.grey[100],
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: _gopeedGreen.withOpacity(0.3)),
      ),
    );

    final focusedPinTheme = defaultPinTheme.copyDecorationWith(
      border: Border.all(color: _gopeedGreen, width: 2),
      borderRadius: BorderRadius.circular(12),
    );

    final submittedPinTheme = defaultPinTheme.copyDecorationWith(
      border: Border.all(color: _gopeedGreen.withOpacity(0.6)),
      borderRadius: BorderRadius.circular(12),
    );

    final errorPinTheme = defaultPinTheme.copyDecorationWith(
      border: Border.all(color: theme.colorScheme.error, width: 2),
    );

    return Scaffold(
      appBar: AppBar(
        title: Text('appLockSetupTitle'.tr),
        centerTitle: true,
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: () => Get.rootDelegate.popRoute(),
        ),
      ),
      body: Center(
        child: SingleChildScrollView(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                Icons.lock_outline,
                size: 80,
                color: _gopeedGreen,
              ),
              const SizedBox(height: 30),
              Obx(() => Text(
                    controller.isConfirming.value
                        ? '${'confirm'.tr} PIN'
                        : 'appLockSetupTitle'.tr,
                    style: theme.textTheme.headlineSmall,
                  )),
              const SizedBox(height: 40),
              Obx(() => controller.isConfirming.value
                  ? Pinput(
                      controller: controller.confirmPinController,
                      length: 4,
                      obscureText: true,
                      obscuringCharacter: '●',
                      showCursor: true,
                      autofocus: true,
                      defaultPinTheme: defaultPinTheme,
                      focusedPinTheme: focusedPinTheme,
                      submittedPinTheme: submittedPinTheme,
                      errorPinTheme: errorPinTheme,
                      onChanged: (value) {
                        if (value.isNotEmpty) controller.resetError();
                      },
                      onCompleted: controller.onPinCompleted,
                    )
                  : Pinput(
                      controller: controller.pinController,
                      length: 4,
                      obscureText: true,
                      obscuringCharacter: '●',
                      showCursor: true,
                      autofocus: true,
                      defaultPinTheme: defaultPinTheme,
                      focusedPinTheme: focusedPinTheme,
                      submittedPinTheme: submittedPinTheme,
                      onCompleted: controller.onPinCompleted,
                    )),
              const SizedBox(height: 20),
              Obx(() {
                if (controller.pinError.value) {
                  return Text(
                    'appLockPinMismatch'.tr,
                    style: TextStyle(color: theme.colorScheme.error),
                  );
                }
                return const SizedBox(height: 16);
              }),
            ],
          ),
        ),
      ),
    );
  }
}
