import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

import 'package:ffi/ffi.dart';

import 'libgopeed_bind.dart';

class LibgopeedWorker {
  LibgopeedWorker() {
    _receivePort.listen(_handleWorkerMessage);
    _errorPort.listen(_handleWorkerError);
    _isolateReady = Isolate.spawn(_workerMain, _receivePort.sendPort, onError: _errorPort.sendPort);
  }

  final ReceivePort _receivePort = ReceivePort();
  final ReceivePort _errorPort = ReceivePort();
  final Map<int, Completer<String>> _pending = {};
  late final Future<Isolate> _isolateReady;
  final Completer<SendPort> _sendPort = Completer<SendPort>();
  int _requestID = 0;

  Future<String> invoke(String method, String path, String query, String body) async {
    await _isolateReady;
    final sendPort = await _sendPort.future;
    final requestID = _requestID++;
    final completer = Completer<String>();
    _pending[requestID] = completer;
    sendPort.send(['invoke', requestID, method, path, query, body]);
    return completer.future;
  }

  Future<String> apiServer(String operation) async {
    await _isolateReady;
    final sendPort = await _sendPort.future;
    final requestID = _requestID++;
    final completer = Completer<String>();
    _pending[requestID] = completer;
    sendPort.send(['apiServer', requestID, operation]);
    return completer.future;
  }

  void _handleWorkerMessage(dynamic message) {
    if (message is SendPort) {
      if (!_sendPort.isCompleted) _sendPort.complete(message);
      return;
    }
    final response = message as List<dynamic>;
    final requestID = response[0] as int;
    final completer = _pending.remove(requestID);
    if (completer == null) return;
    if (response[1] as bool) {
      completer.complete(response[2] as String);
    } else {
      completer.completeError(StateError(response[2] as String));
    }
  }

  void _handleWorkerError(dynamic message) {
    final parts = message is List<dynamic> ? message : <dynamic>[message];
    final error = StateError(parts.map((part) => part.toString()).join('\n'));
    if (!_sendPort.isCompleted) _sendPort.completeError(error);
    for (final completer in _pending.values) {
      completer.completeError(error);
    }
    _pending.clear();
  }

  static void _workerMain(SendPort mainSendPort) {
    final requests = ReceivePort();
    final bindings = LibgopeedBind(DynamicLibrary.open(_libraryName()));
    mainSendPort.send(requests.sendPort);
    requests.listen((dynamic message) {
      final request = message as List<dynamic>;
      final type = request[0] as String;
      final requestID = request[1] as int;
      try {
        final response = switch (type) {
          'invoke' => _invoke(
            bindings,
            request[2] as String,
            request[3] as String,
            request[4] as String,
            request[5] as String,
          ),
          'apiServer' => _apiServer(bindings, request[2] as String),
          _ => throw StateError('Unknown libgopeed worker request: $type'),
        };
        mainSendPort.send([requestID, true, response]);
      } catch (error) {
        mainSendPort.send([requestID, false, error.toString()]);
      }
    });
  }

  static String _apiServer(LibgopeedBind bindings, String operation) {
    final resultPtr = switch (operation) {
      'get' => bindings.GetAPIServerState(),
      'start' => bindings.StartAPIServer(),
      'stop' => bindings.StopAPIServer(),
      'restart' => bindings.RestartAPIServer(),
      _ => throw StateError('Unknown API server operation: $operation'),
    };
    if (resultPtr == nullptr) {
      throw StateError('Gopeed API server operation returned a null response');
    }
    try {
      return resultPtr.cast<Utf8>().toDartString();
    } finally {
      bindings.FreeCString(resultPtr);
    }
  }

  static String _invoke(LibgopeedBind bindings, String method, String path, String query, String body) {
    final methodPtr = method.toNativeUtf8();
    final pathPtr = path.toNativeUtf8();
    final queryPtr = query.toNativeUtf8();
    final bodyPtr = body.toNativeUtf8();
    Pointer<Char>? resultPtr;
    try {
      resultPtr = bindings.Invoke(methodPtr.cast(), pathPtr.cast(), queryPtr.cast(), bodyPtr.cast());
      if (resultPtr == nullptr) {
        throw StateError('Gopeed Invoke returned a null response');
      }
      return resultPtr.cast<Utf8>().toDartString();
    } finally {
      malloc.free(methodPtr);
      malloc.free(pathPtr);
      malloc.free(queryPtr);
      malloc.free(bodyPtr);
      if (resultPtr != null && resultPtr != nullptr) {
        bindings.FreeCString(resultPtr);
      }
    }
  }

  static String _libraryName() {
    if (Platform.isWindows) return 'libgopeed.dll';
    if (Platform.isMacOS) return 'libgopeed.dylib';
    if (Platform.isLinux) return 'libgopeed.so';
    throw UnsupportedError('Desktop FFI is unavailable on ${Platform.operatingSystem}');
  }
}
