import 'dart:async';

import '../common/api_server_state.dart';
import '../common/start_config.dart';
import '../common/task_event.dart';

import '../libgopeed_boot.dart';

LibgopeedBoot create() => LibgopeedBootBrowser();

class LibgopeedBootBrowser implements LibgopeedBoot {
  // do nothing
  @override
  Future<int> start(StartConfig cfg) async => 0;

  @override
  Future<void> stop() async {}

  @override
  Future<ApiServerOperationResult> getApiServerState() async => _unsupportedApiServerResult;

  @override
  Future<ApiServerOperationResult> startApiServer() async => _unsupportedApiServerResult;

  @override
  Future<ApiServerOperationResult> stopApiServer() async => _unsupportedApiServerResult;

  @override
  Future<ApiServerOperationResult> restartApiServer() async => _unsupportedApiServerResult;

  @override
  Future<String> invoke(String method, String path, {String query = '', String body = ''}) {
    throw UnsupportedError('Native Gopeed invoke is unavailable on web');
  }

  @override
  Stream<TaskEvent> get taskEvents => const Stream.empty();

  @override
  Future<void> subscribeTaskEvents(Set<TaskEventType> events) async {}
}

const _unsupportedApiServerResult = ApiServerOperationResult(
  state: ApiServerState(
    enabled: false,
    running: false,
    network: '',
    address: '',
    runningPort: 0,
    pendingApply: false,
    lastError: '',
  ),
  error: '',
);
