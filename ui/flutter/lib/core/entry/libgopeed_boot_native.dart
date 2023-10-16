import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

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
    if (!Util.isDesktop()) {
      _libgopeed = LibgopeedChannel();
    }
  }

  @override
  Future<int> start(StartConfig cfg) async {
    cfg.storage = 'bolt';
    cfg.storageDir = Util.getStorageDir();
    cfg.refreshInterval = 0;
    var port =
        Util.isDesktop() ? await _ffiStart(cfg) : await _libgopeed.start(cfg);
    return port;
  }

  @override
  Future<void> stop() async {
    await _libgopeed.stop();
  }

  _ffiInit() {
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
  }

  // FFI run in isolate
  Future<int> _ffiStart(StartConfig cfg) async {
    return await Isolate.run(() async {
      _ffiInit();
      return await _libgopeed.start(cfg);
    });
  }
}
