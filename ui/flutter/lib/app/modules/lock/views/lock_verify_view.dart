import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:pinput/pinput.dart';
import 'package:gopeed/app/modules/lock/controllers/lock_verify_controller.dart';
import 'package:gopeed/database/database.dart';
import 'package:gopeed/util/util.dart';

const _gopeedGreen = Color(0xFF79C476);

class LockVerifyView extends GetView<LockVerifyController> {
  const LockVerifyView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final hasBiometrics = Database.instance.getBiometricsEnabled();

    // Use fingerprint icon on Android/Linux/Windows, FaceID on macOS/iOS
    final biometricIcon = Util.isIOS() || Util.isMacos() 
        ? Icons.face_retouching_natural
        : Icons.fingerprint;

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
      body: Center(
        child: SingleChildScrollView(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                Icons.lock,
                size: 80,
                color: _gopeedGreen,
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
                      color: _gopeedGreen,
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
