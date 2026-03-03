import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/util/util.dart';
import 'package:local_auth/local_auth.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:gopeed/database/database.dart';
import 'package:gopeed/app/routes/app_pages.dart';

class LockVerifyController extends GetxController {
  final pinController = TextEditingController();
  final pinError = false.obs;

  final LocalAuthentication auth = LocalAuthentication();
  final FlutterSecureStorage secureStorage = const FlutterSecureStorage();

  @override
  void onInit() {
    super.onInit();
    // Try biometrics on start if enabled
    if (Database.instance.getBiometricsEnabled()) {
      checkBiometrics();
    }
  }

  void resetError() {
    if (pinError.value) {
      pinError.value = false;
    }
  }

  Future<void> onPinCompleted(String pin) async {
    final savedPin = await secureStorage.read(key: 'app_lock_pin');
    if (savedPin == pin) {
      // Correct!
      _unlockApp();
    } else {
      pinError.value = true;
      pinController.clear();
    }
  }

  Future<void> checkBiometrics() async {
    bool authenticated = false;
    try {
      authenticated = await auth.authenticate(
        localizedReason: 'appLockUseBiometrics'.tr,
        biometricOnly: true,
        persistAcrossBackgrounding: true,
      );
    } catch (e) {
      // Handle error or just ignore to allow PIN fallback
    }

    if (authenticated) {
      _unlockApp();
    }
  }

  void _unlockApp() {
    // Navigate to HOME directly — do NOT use Routes.ROOT which would
    // rebuild RootView and re-check getAppLockEnabled() → infinite loop
    Get.rootDelegate.offAndToNamed(Routes.HOME);
  }
}
