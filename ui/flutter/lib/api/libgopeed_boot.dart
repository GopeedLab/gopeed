import "libgopeed_boot_stub.dart"
    if (dart.library.html) 'entry/libgopeed_boot_browser.dart'
    if (dart.library.io) 'entry/libgopeed_boot_native.dart';
import 'native/libgopeed_interface.dart';

abstract class LibgopeedBoot implements LibgopeedApi {
  static LibgopeedBoot? _instance;

  static LibgopeedBoot get instance {
    _instance ??= LibgopeedBoot();
    return _instance!;
  }

  factory LibgopeedBoot() => create();
}
