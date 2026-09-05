import 'dart:async';
import 'dart:ffi';
import 'dart:io';

import '../../util/util.dart';
import '../common/libgopeed_channel.dart';
import '../common/libgopeed_ffi.dart';
import '../common/libgopeed_interface.dart';
import '../common/start_config.dart';
import '../ffi/libgopeed_bind.dart';
import '../libgopeed_boot.dart';

LibgopeedBoot create() => LibgopeedBootNative();

class LibgopeedBootNative implements LibgopeedBoot {
  late LibgopeedInterface _libgopeed;

  LibgopeedBootNative() {
    if (Util.isDesktop()) {
      var libName = "libgopeed.";
      if (Platform.isWindows) {
        libName += "dll";
      }
      if (Platform.isMacOS) {
        libName += "dylib";
      }
      if (Platform.isLinux) {
        libName += "so";
      }
      _libgopeed = LibgopeedFFi(LibgopeedBind(DynamicLibrary.open(libName)));
    } else {
      _libgopeed = LibgopeedChannel();
    }
  }

  @override
  Future<int> start(StartConfig cfg) async {
    cfg.storage = 'bolt';
    cfg.storageDir = Util.getStorageDir();
    cfg.refreshInterval = 0;
    var port = await _libgopeed.start(cfg);
    return port;
  }

  @override
  Future<void> stop() async {
    await _libgopeed.stop();
  }
}
