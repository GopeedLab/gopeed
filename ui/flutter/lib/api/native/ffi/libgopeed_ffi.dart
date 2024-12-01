import 'dart:async';

import 'package:ffi/ffi.dart';

import '../libgopeed_interface.dart';
import 'libgopeed_bind.dart';

class LibgopeedFFi implements LibgopeedAbi {
  late LibgopeedBind _libgopeed;

  LibgopeedFFi(LibgopeedBind libgopeed) {
    _libgopeed = libgopeed;
  }

  @override
  Future<String> create(String cfg) {
    final cfgPtr = cfg.toNativeUtf8();
    try {
      final result = _libgopeed.Create(cfgPtr.cast());
      return Future.value(result.cast<Utf8>().toDartString());
    } finally {
      calloc.free(cfgPtr);
    }
  }

  @override
  Future<String> invoke(int instance, String params) {
    final paramsPtr = params.toNativeUtf8();
    try {
      final result = _libgopeed.Invoke(instance, paramsPtr.cast());
      return Future.value(result.cast<Utf8>().toDartString());
    } finally {
      calloc.free(paramsPtr);
    }
  }
}
