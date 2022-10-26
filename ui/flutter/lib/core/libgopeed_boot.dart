import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

import 'package:gopeed/core/common/libgopeed_channel.dart';
import 'package:gopeed/core/common/libgopeed_ffi.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:path_provider/path_provider.dart';

import '../util/util.dart';
import 'common/libgopeed_interface.dart';
import 'ffi/libgopeed_bind.dart';

const unixSocketPath = 'gopeed.sock';

class LibgopeedBoot {
  String network = "";
  String address = "";
  int port = 0;
  int refreshInterval = 250;

  late LibgopeedInterface _libgopeed;
  late SendPort _childSendPort;

  static LibgopeedBoot? _instance;

  static LibgopeedBoot get instance {
    if (_instance == null) {
      _instance = LibgopeedBoot._internal();
      if (!Util.isDesktop()) {
        _instance!._libgopeed = LibgopeedChannel();
      }
    }
    return _instance!;
  }

  LibgopeedBoot._internal();

  Future<void> start() async {
    var storageDir = "./";
    if (!Util.isUnix()) {
      // not support unix socket, use tcp
      network = "tcp";
      address = "127.0.0.1:0";
    } else {
      network = "unix";
      if (Util.isDesktop()) {
        address = unixSocketPath;
      } else if (Platform.isAndroid) {
        address = "${(await getTemporaryDirectory()).path}/$unixSocketPath";
        storageDir = (await getExternalStorageDirectory())?.path ?? "";
      }
    }
    final cfg = StartConfig(
        network: network,
        address: address,
        storage: 'bolt',
        storageDir: storageDir,
        refreshInterval: refreshInterval);
    if (Util.isDesktop()) {
      port = await _ffiStart(cfg);
    } else {
      port = await _libgopeed.start(cfg);
    }
  }

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
