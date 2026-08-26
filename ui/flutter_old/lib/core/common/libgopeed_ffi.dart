import 'dart:async';
import 'dart:convert';
import 'dart:ffi';

import 'package:ffi/ffi.dart';

import '../ffi/libgopeed_bind.dart';
import 'libgopeed_interface.dart';
import 'start_config.dart';

class LibgopeedFFi implements LibgopeedInterface {
  late LibgopeedBind _libgopeed;

  LibgopeedFFi(LibgopeedBind libgopeed) {
    _libgopeed = libgopeed;
  }

  @override
  Future<int> start(StartConfig cfg) {
    var completer = Completer<int>();
    var result = _libgopeed.Start(jsonEncode(cfg).toNativeUtf8().cast());
    if (result.r1 != nullptr) {
      completer.completeError(Exception(result.r1.cast<Utf8>().toDartString()));
    } else {
      completer.complete(result.r0);
    }
    return completer.future;
  }

  @override
  Future<void> stop() {
    var completer = Completer<void>();
    _libgopeed.Stop();
    completer.complete();
    return completer.future;
  }
}
