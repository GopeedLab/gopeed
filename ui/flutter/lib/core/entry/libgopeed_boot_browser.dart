import 'dart:async';

import '../libgopeed_boot.dart';

LibgopeedBoot create() => LibgopeedBootBrowser();

class LibgopeedBootBrowser implements LibgopeedBoot {
  // do nothing
  @override
  Future<int> start(String network, String address) async {
    return 0;
  }

  @override
  Future<void> stop() async {}
}
