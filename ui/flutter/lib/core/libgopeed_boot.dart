import 'common/api_server_state.dart';
import 'common/start_config.dart';
import 'common/task_event.dart';
import "libgopeed_boot_stub.dart"
    if (dart.library.html) 'entry/libgopeed_boot_browser.dart'
    if (dart.library.io) 'entry/libgopeed_boot_native.dart';

abstract class LibgopeedBoot {
  static LibgopeedBoot? _instance;

  static LibgopeedBoot get instance {
    _instance ??= LibgopeedBoot();
    return _instance!;
  }

  factory LibgopeedBoot() => create();

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
