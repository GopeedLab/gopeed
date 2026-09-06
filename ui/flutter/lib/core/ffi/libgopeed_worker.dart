import 'dart:async';
import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

import 'package:ffi/ffi.dart';

import 'libgopeed_bind.dart';

typedef _InvokeResultNative = Void Function(Uint64 requestID, Int32 success, Pointer<Char> payload);
typedef _TaskEventNative = Void Function(Pointer<Char> payload);

class LibgopeedWorker {
  factory LibgopeedWorker() => _instance;

  LibgopeedWorker._() {
    _receivePort.listen(_handleWorkerMessage);
    _errorPort.listen(_handleWorkerError);
    _isolateReady = Isolate.spawn(_workerMain, _receivePort.sendPort, onError: _errorPort.sendPort);
  }

  static final LibgopeedWorker _instance = LibgopeedWorker._();

  final ReceivePort _receivePort = ReceivePort();
  final ReceivePort _errorPort = ReceivePort();
  final Map<int, Completer<Object?>> _pending = {};
  final StreamController<Map<String, dynamic>> _taskEvents = StreamController<Map<String, dynamic>>.broadcast();
  late final Future<Isolate> _isolateReady;
  final Completer<SendPort> _sendPort = Completer<SendPort>();
  Future<void>? _stopFuture;
  int _requestID = 0;
  bool _stopping = false;
  bool _stopped = false;

  Stream<Map<String, dynamic>> get taskEvents => _taskEvents.stream;

  Future<int> start(String config) => _request<int>('start', [config]);

  Future<void> stop() => _stopFuture ??= _stop();

  Future<void> _stop() async {
    _stopping = true;
    // The native bridge owns the bounded drain. Sending Stop immediately is
    // important because an InvokeAsync handler such as Resolve may never return.
    await _request<Object?>('stop', const [], allowWhileStopping: true);
    _stopped = true;
    final stoppingError = StateError('libgopeed worker stopped before the request completed');
    for (final completer in _pending.values) {
      completer.completeError(stoppingError);
    }
    _pending.clear();
    await _taskEvents.close();
    _receivePort.close();
    _errorPort.close();
  }

  Future<Object?> invoke(String method, String path, String query, String body) {
    return _request<Object?>('invoke', [method, path, query, body]);
  }

  Future<String> apiServer(String operation) => _request<String>('apiServer', [operation]);

  Future<void> subscribeTaskEvents(int mask) async {
    await _request<Object?>('subscribeTaskEvents', [mask]);
  }

  Future<T> _request<T>(String type, List<Object?> arguments, {bool allowWhileStopping = false}) async {
    if (_stopped || (_stopping && !allowWhileStopping)) {
      throw StateError('libgopeed worker is stopping');
    }
    await _isolateReady;
    final sendPort = await _sendPort.future;
    final requestID = _requestID++;
    final completer = Completer<Object?>();
    _pending[requestID] = completer;
    sendPort.send([type, requestID, ...arguments]);
    return (await completer.future) as T;
  }

  void _handleWorkerMessage(dynamic message) {
    if (message is SendPort) {
      if (!_sendPort.isCompleted) _sendPort.complete(message);
      return;
    }
    final response = message as List<dynamic>;
    switch (response[0] as String) {
      case 'response':
        final requestID = response[1] as int;
        final completer = _pending.remove(requestID);
        if (completer == null) return;
        if (response[2] as bool) {
          completer.complete(response[3]);
        } else {
          completer.completeError(StateError(response[3] as String));
        }
      case 'taskEvent':
        _taskEvents.add((response[1] as Map<Object?, Object?>).cast<String, dynamic>());
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
    _taskEvents.addError(error);
  }

  static void _workerMain(SendPort mainSendPort) {
    final requests = ReceivePort();
    final bindings = LibgopeedBind(DynamicLibrary.open(_libraryName()));
    late final NativeCallable<_InvokeResultNative> invokeResultCallback;
    late final NativeCallable<_TaskEventNative> taskEventCallback;

    invokeResultCallback = NativeCallable<_InvokeResultNative>.listener((
      int requestID,
      int success,
      Pointer<Char> payload,
    ) {
      if (payload == nullptr) {
        mainSendPort.send(['response', requestID, false, 'Gopeed InvokeAsync returned a null response']);
        return;
      }
      try {
        final result = payload.cast<Utf8>().toDartString();
        mainSendPort.send(['response', requestID, success != 0, success != 0 ? jsonDecode(result) : result]);
      } catch (error) {
        mainSendPort.send(['response', requestID, false, error.toString()]);
      } finally {
        bindings.FreeCString(payload);
      }
    });

    taskEventCallback = NativeCallable<_TaskEventNative>.listener((Pointer<Char> payload) {
      if (payload == nullptr) return;
      try {
        final event = jsonDecode(payload.cast<Utf8>().toDartString()) as Map<String, dynamic>;
        mainSendPort.send(['taskEvent', event]);
      } finally {
        bindings.FreeCString(payload);
      }
    });

    mainSendPort.send(requests.sendPort);
    requests.listen((dynamic message) {
      final request = message as List<dynamic>;
      final type = request[0] as String;
      final requestID = request[1] as int;
      if (type == 'invoke') {
        try {
          _invokeAsync(
            bindings,
            invokeResultCallback.nativeFunction.address,
            requestID,
            request[2] as String,
            request[3] as String,
            request[4] as String,
            request[5] as String,
          );
        } catch (error) {
          mainSendPort.send(['response', requestID, false, error.toString()]);
        }
        return;
      }

      try {
        final response = switch (type) {
          'start' => _start(bindings, request[2] as String),
          'stop' => _stopNative(bindings),
          'apiServer' => _apiServer(bindings, request[2] as String),
          'subscribeTaskEvents' => _subscribeTaskEvents(
            bindings,
            taskEventCallback.nativeFunction.address,
            request[2] as int,
          ),
          _ => throw StateError('Unknown libgopeed worker request: $type'),
        };
        mainSendPort.send(['response', requestID, true, response]);
        if (type == 'stop') {
          invokeResultCallback.close();
          taskEventCallback.close();
          requests.close();
        }
      } catch (error) {
        mainSendPort.send(['response', requestID, false, error.toString()]);
      }
    });
  }

  static int _start(LibgopeedBind bindings, String config) {
    final configPtr = config.toNativeUtf8();
    try {
      final result = bindings.Start(configPtr.cast());
      if (result.r1 != nullptr) {
        try {
          throw StateError(result.r1.cast<Utf8>().toDartString());
        } finally {
          bindings.FreeCString(result.r1);
        }
      }
      return result.r0;
    } finally {
      malloc.free(configPtr);
    }
  }

  static Object? _stopNative(LibgopeedBind bindings) {
    bindings.Stop();
    return null;
  }

  static Object? _subscribeTaskEvents(LibgopeedBind bindings, int callback, int mask) {
    bindings.SubscribeTaskEvents(mask, mask == 0 ? 0 : callback);
    return null;
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

  static void _invokeAsync(
    LibgopeedBind bindings,
    int callback,
    int requestID,
    String method,
    String path,
    String query,
    String body,
  ) {
    final methodPtr = method.toNativeUtf8();
    final pathPtr = path.toNativeUtf8();
    final queryPtr = query.toNativeUtf8();
    final bodyPtr = body.toNativeUtf8();
    try {
      bindings.InvokeAsync(methodPtr.cast(), pathPtr.cast(), queryPtr.cast(), bodyPtr.cast(), requestID, callback);
    } finally {
      malloc.free(methodPtr);
      malloc.free(pathPtr);
      malloc.free(queryPtr);
      malloc.free(bodyPtr);
    }
  }

  static String _libraryName() {
    if (Platform.isWindows) return 'libgopeed.dll';
    if (Platform.isMacOS) return 'libgopeed.dylib';
    if (Platform.isLinux) return 'libgopeed.so';
    throw UnsupportedError('Desktop FFI is unavailable on ${Platform.operatingSystem}');
  }
}
