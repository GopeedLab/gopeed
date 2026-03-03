import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/util/util.dart';
import 'package:local_auth/local_auth.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:gopeed/database/database.dart';

class LockSetupController extends GetxController {
  final pinController = TextEditingController();
  final confirmPinController = TextEditingController();
  
  final isConfirming = false.obs;
  final pinError = false.obs;
  String _firstPin = '';

  final LocalAuthentication auth = LocalAuthentication();
  final FlutterSecureStorage secureStorage = const FlutterSecureStorage();

  void onPinCompleted(String pin) {
    if (!isConfirming.value) {
      // First step done, save and move to confirmation
      _firstPin = pin;
      isConfirming.value = true;
      pinController.clear();
    } else {
      // Confirmation step — compare with the first PIN
      if (_firstPin == pin) {
        _savePinAndFinish(pin);
      } else {
        pinError.value = true;
        confirmPinController.clear();
      }
    }
  }

  void resetError() {
    if (pinError.value) {
      pinError.value = false;
    }
  }

  Future<void> _savePinAndFinish(String pin) async {
    // 1. Save PIN to secure storage
    await secureStorage.write(key: 'app_lock_pin', value: pin);
    
    // 2. Mark app lock as enabled in fast Database
    Database.instance.setAppLockEnabled(true);

    // 3. Check for Biometrics
    bool canCheckBiometrics = false;
    try {
      if (Util.isMobile() || Util.isMacos() || Util.isWindows()) {
         canCheckBiometrics = await auth.canCheckBiometrics || await auth.isDeviceSupported();
      }
    } catch (e) {
      canCheckBiometrics = false;
    }

    if (canCheckBiometrics) {
      // Ask user to use biometrics
      final useBiometrics = await Get.dialog<bool>(
        AlertDialog(
          title: Text('appLockSettingsTitle'.tr),
          content: Text('appLockUseBiometrics'.tr),
          actions: [
            TextButton(
              onPressed: () => Get.back(result: false),
              child: Text('cancel'.tr),
            ),
            TextButton(
              onPressed: () => Get.back(result: true),
              child: Text('confirm'.tr),
            ),
          ],
        ),
      );

      if (useBiometrics == true) {
        // Authenticate once to verify and save choice
        try {
          final didAuthenticate = await auth.authenticate(
            localizedReason: 'appLockUseBiometrics'.tr,
            biometricOnly: true,
          );
          if (didAuthenticate) {
            Database.instance.setBiometricsEnabled(true);
          }
        } catch (e) {
          // Ignore
        }
      }
    }

    // Go back to previous screen (Settings)
    Get.back(result: true);
  }
}
