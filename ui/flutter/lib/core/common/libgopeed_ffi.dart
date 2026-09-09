import 'dart:async';
import 'dart:convert';

import '../ffi/libgopeed_worker.dart';
import 'api_server_state.dart';
import 'libgopeed_interface.dart';
import 'start_config.dart';
import 'task_event.dart';

class LibgopeedFFi implements LibgopeedInterface {
  LibgopeedFFi() {
    _worker.taskEvents.listen(_onTaskEvent, onError: _taskEvents.addError);
  }

  final _taskEvents = StreamController<TaskEvent>.broadcast();
  final _worker = LibgopeedWorker();

  void _onTaskEvent(Map<String, dynamic> payload) {
    _taskEvents.add(TaskEvent.fromJson(payload));
  }

  @override
  Future<int> start(StartConfig cfg) => _worker.start(jsonEncode(cfg));

  @override
  Future<void> stop() => _worker.stop();

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
    return jsonEncode(await _worker.invoke(method, path, query, body));
  }

  @override
  Stream<TaskEvent> get taskEvents => _taskEvents.stream;

  @override
  Future<void> subscribeTaskEvents(Set<TaskEventType> events) => _worker.subscribeTaskEvents(events.mask);
}
