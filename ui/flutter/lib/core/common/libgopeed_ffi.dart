import 'dart:async';
import 'dart:convert';
import 'dart:ffi';

import 'package:ffi/ffi.dart';

import '../ffi/libgopeed_bind.dart';
import '../ffi/libgopeed_worker.dart';
import 'api_server_state.dart';
import 'libgopeed_interface.dart';
import 'start_config.dart';
import 'task_event.dart';

class LibgopeedFFi implements LibgopeedInterface {
  late LibgopeedBind _libgopeed;
  final _taskEvents = StreamController<TaskEvent>.broadcast();
  final _worker = LibgopeedWorker();
  late final NativeCallable<Void Function(Pointer<Char>)> _taskEventCallback;

  LibgopeedFFi(LibgopeedBind libgopeed) {
    _libgopeed = libgopeed;
    _taskEventCallback = NativeCallable<Void Function(Pointer<Char>)>.listener(_onTaskEvent);
  }

  void _onTaskEvent(Pointer<Char> payload) {
    try {
      final decoded = jsonDecode(payload.cast<Utf8>().toDartString()) as Map<String, dynamic>;
      _taskEvents.add(TaskEvent.fromJson(decoded));
    } finally {
      _libgopeed.FreeCString(payload);
    }
  }

  @override
  Future<int> start(StartConfig cfg) {
    var completer = Completer<int>();
    final cfgPtr = jsonEncode(cfg).toNativeUtf8();
    try {
      final result = _libgopeed.Start(cfgPtr.cast());
      if (result.r1 != nullptr) {
        final message = result.r1.cast<Utf8>().toDartString();
        _libgopeed.FreeCString(result.r1);
        completer.completeError(Exception(message));
      } else {
        completer.complete(result.r0);
      }
    } finally {
      malloc.free(cfgPtr);
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

  Future<ApiServerOperationResult> _apiServerOperation(String operation) async {
    final payload = await _worker.apiServer(operation);
    return ApiServerOperationResult.fromJson(jsonDecode(payload) as Map<String, dynamic>);
  }

  @override
  Future<ApiServerOperationResult> getApiServerState() => _apiServerOperation('get');

  @override
  Future<ApiServerOperationResult> startApiServer() => _apiServerOperation('start');

  @override
  Future<ApiServerOperationResult> stopApiServer() => _apiServerOperation('stop');

  @override
  Future<ApiServerOperationResult> restartApiServer() => _apiServerOperation('restart');

  @override
  Future<String> invoke(String method, String path, {String query = '', String body = ''}) async {
    final methodPtr = method.toNativeUtf8();
    final pathPtr = path.toNativeUtf8();
    final queryPtr = query.toNativeUtf8();
    final bodyPtr = body.toNativeUtf8();
    Pointer<Char>? resultPtr;
    try {
      resultPtr = _libgopeed.Invoke(methodPtr.cast(), pathPtr.cast(), queryPtr.cast(), bodyPtr.cast());
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
        _libgopeed.FreeCString(resultPtr);
      }
    }
  }

  @override
  Stream<TaskEvent> get taskEvents => _taskEvents.stream;

  @override
  Future<void> subscribeTaskEvents(Set<TaskEventType> events) async {
    _libgopeed.SubscribeTaskEvents(events.mask, events.isEmpty ? 0 : _taskEventCallback.nativeFunction.address);
  }
}
