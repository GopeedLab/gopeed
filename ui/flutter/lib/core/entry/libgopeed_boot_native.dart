import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

import 'package:path_provider/path_provider.dart';

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
  late SendPort _childSendPort;

  LibgopeedBootNative() {
    if (!Util.isDesktop()) {
      _libgopeed = LibgopeedChannel();
    }
  }

  @override
  Future<int> start(String network, String address) async {
    var storageDir = "./";
    if (Platform.isAndroid) {
      storageDir = (await getExternalStorageDirectory())?.path ?? storageDir;
    }
    if (Platform.isIOS) {
      storageDir = (await getLibraryDirectory()).path;
    }
    final cfg = StartConfig(
        network: network,
        address: address,
        storage: 'bolt',
        storageDir: storageDir,
        refreshInterval: 0);
    var port =
        Util.isDesktop() ? await _ffiStart(cfg) : await _libgopeed.start(cfg);
    return port;
  }

  @override
  Future<void> stop() async {
    if (Util.isDesktop()) {
      await _ffiStop();
    } else {
      await _libgopeed.stop();
    }
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
    var startCompleter = Completer<int>();

    var mainReceive = ReceivePort();
    var isolate = await Isolate.spawn((SendPort sendPort) {
      _ffiInit();
      var childReceive = ReceivePort();
      sendPort.send(childReceive.sendPort);

      childReceive.listen((message) async {
        late _Result result;
        try {
          if (message is _StartParam) {
            result = _StartResult();
            (result as _StartResult).port = await _libgopeed.start(message.cfg);
          }
          if (message is _StopParam) {
            result = _StopResult();
            await _libgopeed.stop();
            childReceive.close();
            Isolate.exit(sendPort);
          }
        } on Exception catch (e) {
          result.exception = e;
        }
        sendPort.send(result);
      });
    }, mainReceive.sendPort);

    var childCompleter = Completer<void>();
    mainReceive.listen((message) {
      if (message is SendPort) {
        _childSendPort = message;
        childCompleter.complete();
      }
      if (message is _StartResult) {
        if (message.exception != null) {
          startCompleter.completeError(message.exception!);
        } else {
          startCompleter.complete(message.port!);
        }
      }
      if (message is _StopResult) {
        isolate.kill();
        if (message.exception != null) {
          _stopCompleter.completeError(message.exception!);
        } else {
          _stopCompleter.complete();
        }
      }
    });
    await childCompleter.future;
    _childSendPort.send(_StartParam(cfg: cfg));
    return startCompleter.future;
  }

  late Completer<void> _stopCompleter;

  Future<void> _ffiStop() async {
    _stopCompleter = Completer<void>();
    _childSendPort.send(_StopParam());
    return _stopCompleter.future;
  }
}

class _Result {
  Exception? exception;

  _Result();
}

class _StartParam {
  final StartConfig cfg;

  _StartParam({required this.cfg});
}

class _StartResult extends _Result {
  int? port;

  _StartResult();
}

class _StopParam {}

class _StopResult extends _Result {
  _StopResult();
}
