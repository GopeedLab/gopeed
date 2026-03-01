import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:pinput/pinput.dart';
import 'package:gopeed/app/modules/lock/controllers/lock_verify_controller.dart';
import 'package:gopeed/database/database.dart';
import 'package:gopeed/util/util.dart';

class LockVerifyView extends GetView<LockVerifyController> {
  const LockVerifyView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    // If not locked and got here by accident, just pop back
    final theme = Theme.of(context);
    final hasBiometrics = Database.instance.getBiometricsEnabled();

    // Use fingerprint icon on Android/Linux/Windows, FaceID on macOS/iOS (simplified)
    final biometricIcon = Util.isIOS() || Util.isMacos() 
        ? Icons.face_retouching_natural
        : Icons.fingerprint;

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
      body: Center(
        child: SingleChildScrollView(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                Icons.lock,
                size: 80,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(height: 30),
              Text(
                'appLockVerifyTitle'.tr,
                style: theme.textTheme.headlineSmall,
              ),
              const SizedBox(height: 40),
              Pinput(
                controller: controller.pinController,
                length: 4,
                obscureText: true,
                autofocus: true,
                defaultPinTheme: defaultPinTheme,
                focusedPinTheme: focusedPinTheme,
                errorPinTheme: errorPinTheme,
                onChanged: (_) => controller.resetError(),
                onCompleted: controller.onPinCompleted,
              ),
              const SizedBox(height: 20),
              Obx(() {
                if (controller.pinError.value) {
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 20),
                    child: Text(
                      'appLockPinMismatch'.tr,
                      style: TextStyle(color: theme.colorScheme.error),
                    ),
                  );
                }
                return const SizedBox(height: 36);
              }),
              if (hasBiometrics)
                InkWell(
                  onTap: () => controller.checkBiometrics(),
                  borderRadius: BorderRadius.circular(50),
                  child: Padding(
                    padding: const EdgeInsets.all(12.0),
                    child: Icon(
                      biometricIcon,
                      size: 60,
                      color: theme.colorScheme.primary,
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
