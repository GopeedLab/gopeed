import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:pinput/pinput.dart';
import 'package:gopeed/app/modules/lock/controllers/lock_setup_controller.dart';
import 'package:gopeed/util/util.dart';

class LockSetupView extends GetView<LockSetupController> {
  const LockSetupView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    
    final defaultPinTheme = PinTheme(
      width: 56,
      height: 56,
      textStyle: const TextStyle(
        fontSize: 22,
        color: Color.fromRGBO(30, 60, 87, 1),
      ),
      decoration: BoxDecoration(
        color: theme.scaffoldBackgroundColor,
        borderRadius: BorderRadius.circular(19),
        border: Border.all(color: theme.colorScheme.primary.withOpacity(0.3)),
      ),
    );

    final focusedPinTheme = defaultPinTheme.copyDecorationWith(
      border: Border.all(color: theme.colorScheme.primary),
      borderRadius: BorderRadius.circular(8),
    );

    final errorPinTheme = defaultPinTheme.copyDecorationWith(
      border: Border.all(color: theme.colorScheme.error),
    );

    return Scaffold(
      appBar: AppBar(
        title: Text('appLockSetupTitle'.tr),
        centerTitle: true,
      ),
      body: Center(
        child: SingleChildScrollView(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                Icons.lock_outline,
                size: 80,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(height: 30),
              Obx(() => Text(
                    controller.isConfirming.value
                        ? 'confirm'.tr + ' PIN'
                        : 'appLockSetupTitle'.tr,
                    style: theme.textTheme.headlineSmall,
                  )),
              const SizedBox(height: 40),
              Obx(() => controller.isConfirming.value
                  ? Pinput(
                      controller: controller.confirmPinController,
                      length: 4,
                      obscureText: true,
                      autofocus: true,
                      defaultPinTheme: defaultPinTheme,
                      focusedPinTheme: focusedPinTheme,
                      errorPinTheme: errorPinTheme,
                      onChanged: (_) => controller.resetError(),
                      onCompleted: controller.onPinCompleted,
                    )
                  : Pinput(
                      controller: controller.pinController,
                      length: 4,
                      obscureText: true,
                      autofocus: true,
                      defaultPinTheme: defaultPinTheme,
                      focusedPinTheme: focusedPinTheme,
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
