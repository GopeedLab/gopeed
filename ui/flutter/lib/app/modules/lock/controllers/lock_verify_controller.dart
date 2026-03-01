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
        stickyAuth: true,
      );
    } catch (e) {
      // Handle error or just ignore to allow PIN fallback
    }

    if (authenticated) {
      _unlockApp();
    }
  }

  void _unlockApp() {
    // Navigate back to the previous context, or to RootView.
    // If we presented this screen over everything, we can just Get.offAllNamed(Routes.ROOT);
    // Setting `isLocked` state in AppController will also help.
    Get.offAllNamed(Routes.ROOT);
  }
}
