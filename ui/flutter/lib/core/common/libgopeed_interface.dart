import 'api_server_state.dart';
import 'start_config.dart';
import 'task_event.dart';

abstract class LibgopeedInterface {
  Future<int> start(StartConfig cfg);

  Future<void> stop();

  Future<ApiServerOperationResult> getApiServerState();

  Future<ApiServerOperationResult> startApiServer();

  Future<ApiServerOperationResult> stopApiServer();

  Future<ApiServerOperationResult> restartApiServer();

  Future<String> invoke(String method, String path, {String query = '', String body = ''});

  Stream<TaskEvent> get taskEvents;

  Future<void> subscribeTaskEvents(Set<TaskEventType> events);
}
