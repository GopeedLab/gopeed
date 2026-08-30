import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';

import 'api_server_state.dart';
import 'libgopeed_interface.dart';
import 'start_config.dart';
import 'task_event.dart';

class LibgopeedChannel implements LibgopeedInterface {
  static const _channel = MethodChannel('gopeed.com/libgopeed');
  final _taskEvents = StreamController<TaskEvent>.broadcast();

  LibgopeedChannel() {
    _channel.setMethodCallHandler((call) async {
      if (call.method != 'taskEvent') return;
      final payload = jsonDecode(call.arguments as String) as Map<String, dynamic>;
      _taskEvents.add(TaskEvent.fromJson(payload));
    });
  }

  @override
  Future<int> start(StartConfig cfg) async {
    final port = await _channel.invokeMethod('start', {'cfg': jsonEncode(cfg)});
    return port as int;
  }

  @override
  Future<void> stop() async {
    return await _channel.invokeMethod('stop');
  }

  Future<ApiServerOperationResult> _apiServerOperation(String method) async {
    final payload = await _channel.invokeMethod<String>(method) ?? '';
    return ApiServerOperationResult.fromJson(jsonDecode(payload) as Map<String, dynamic>);
  }

  @override
  Future<ApiServerOperationResult> getApiServerState() => _apiServerOperation('getApiServerState');

  @override
  Future<ApiServerOperationResult> startApiServer() => _apiServerOperation('startApiServer');

  @override
  Future<ApiServerOperationResult> stopApiServer() => _apiServerOperation('stopApiServer');

  @override
  Future<ApiServerOperationResult> restartApiServer() => _apiServerOperation('restartApiServer');

  @override
  Future<String> invoke(String method, String path, {String query = '', String body = ''}) async {
    final result = await _channel.invokeMethod<String>('invoke', {
      'method': method,
      'path': path,
      'query': query,
      'body': body,
    });
    return result ?? '';
  }

  @override
  Stream<TaskEvent> get taskEvents => _taskEvents.stream;

  @override
  Future<void> subscribeTaskEvents(Set<TaskEventType> events) {
    return _channel.invokeMethod<void>('subscribeTaskEvents', {'mask': events.mask});
  }
}
