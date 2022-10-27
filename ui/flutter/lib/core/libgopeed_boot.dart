import "./libgopeed_boot_stub.dart"
    if (dart.library.html) 'entry/libgopeed_boot_browser.dart'
    if (dart.library.io) 'entry/libgopeed_boot_native.dart';

class LibgopeedConfig {
  late String network;
  late String address;
  late int refreshInterval;

  LibgopeedConfig({
    this.network = "tcp",
    this.address = "127.0.0.1:9999",
    this.refreshInterval = 500,
  });
}

abstract class LibgopeedBoot {
  static const unixSocketPath = 'gopeed.sock';

  static LibgopeedBoot? _instance;

  static LibgopeedBoot get instance {
    _instance ??= LibgopeedBoot();
    return _instance!;
  }

  factory LibgopeedBoot() => create();

  Future<void> start();

  Future<void> stop();

  LibgopeedConfig get config;
}
