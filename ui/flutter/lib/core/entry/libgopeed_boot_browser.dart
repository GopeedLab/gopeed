import 'dart:async';

import '../libgopeed_boot.dart';

LibgopeedBoot create() => LibgopeedBootBrowser();

class LibgopeedBootBrowser implements LibgopeedBoot {
  late LibgopeedConfig _config;

  LibgopeedBootBrowser() {
    _config = LibgopeedConfig();
  }

  @override
  LibgopeedConfig get config => _config;

  @override
  Future<void> start() async {}

  @override
  Future<void> stop() async {}
}
