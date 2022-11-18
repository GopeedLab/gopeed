import 'dart:async';

import 'package:gopeed/core/common/start_config.dart';

import '../libgopeed_boot.dart';

LibgopeedBoot create() => LibgopeedBootBrowser();

class LibgopeedBootBrowser implements LibgopeedBoot {
  // do nothing
  @override
  Future<int> start(StartConfig cfg) async {
    return 0;
  }

  @override
  Future<void> stop() async {}
}
