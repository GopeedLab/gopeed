import "libgopeed_boot_stub.dart"
    if (dart.library.html) 'entry/libgopeed_boot_browser.dart'
    if (dart.library.io) 'entry/libgopeed_boot_native.dart';
import 'native/libgopeed_interface.dart';

abstract class LibgopeedBoot implements LibgopeedApiSingleton {
  static LibgopeedApi? _instance;

  static void singleton(LibgopeedApi instance) {
    _instance ??= instance;
  }

  static LibgopeedApi get instance {
    return _instance!;
  }

  factory LibgopeedBoot() => create();
}
