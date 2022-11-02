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
  late LibgopeedConfig _config;
  late LibgopeedInterface _libgopeed;
  late SendPort _childSendPort;

  LibgopeedBootNative() {
    _config = LibgopeedConfig();
    if (!Util.isDesktop()) {
      _libgopeed = LibgopeedChannel();
    }
  }

  @override
  LibgopeedConfig get config => _config;

  @override
  Future<void> start() async {
    var storageDir = "./";
    if (!Util.isUnix()) {
      // not support unix socket, use tcp
      _config.network = "tcp";
      _config.address = "127.0.0.1:0";
    } else {
      _config.network = "unix";
      if (Util.isDesktop()) {
        _config.address = LibgopeedBoot.unixSocketPath;
      } else if (Platform.isAndroid) {
        _config.address =
            "${(await getTemporaryDirectory()).path}/${LibgopeedBoot.unixSocketPath}";
        storageDir = (await getExternalStorageDirectory())?.path ?? "";
      }
    }
    final cfg = StartConfig(
        network: _config.network,
        address: _config.address,
        storage: 'bolt',
        storageDir: storageDir,
        refreshInterval: _config.refreshInterval);
    var port =
        Util.isDesktop() ? await _ffiStart(cfg) : await _libgopeed.start(cfg);
    if (_config.network == "tcp") {
      _config.address = "${_config.address.split(":")[0]}:$port";
    }
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
    if (Platform.isMacOS) {
      _libgopeed =
          LibgopeedFFi(LibgopeedBind(DynamicLibrary.open('libgopeed.dylib')));
    }
    if (Platform.isLinux) {
      _libgopeed =
          LibgopeedFFi(LibgopeedBind(DynamicLibrary.open('libgopeed.so')));
    }
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
