import 'dart:async';
import 'dart:ffi';

import 'package:ffi/ffi.dart';

import '../libgopeed_interface.dart';
import 'libgopeed_bind.dart';

class LibgopeedFFi implements LibgopeedAbi {
  late LibgopeedBind _libgopeed;

  LibgopeedFFi(LibgopeedBind libgopeed) {
    _libgopeed = libgopeed;
  }

  @override
  Future<void> init(String cfg) {
    final cfgPtr = cfg.toNativeUtf8();
    try {
      final result = _libgopeed.Init(cfgPtr.cast());
      if (result != nullptr) {
        throw Exception(
            'Libgopeed init failed: ${result.cast<Utf8>().toDartString()}');
      }
      return Future.value();
    } finally {
      calloc.free(cfgPtr);
    }
  }

  @override
  Future<String> invoke(String params) {
    final paramsPtr = params.toNativeUtf8();
    try {
      final result = _libgopeed.Invoke(paramsPtr.cast());
      return Future.value(result.cast<Utf8>().toDartString());
    } finally {
      calloc.free(paramsPtr);
    }
  }
}
