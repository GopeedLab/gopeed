import 'package:dio/dio.dart';

import '../model/create_task.dart';
import '../model/create_task_batch.dart';
import '../model/downloader_config.dart';
import '../model/extension.dart';
import '../model/http_listen_result.dart';
import '../model/info.dart';
import '../model/install_extension.dart';
import '../model/request.dart';
import '../model/resolve_result.dart';
import '../model/switch_extension.dart';
import '../model/task.dart';
import '../model/task_filter.dart';
import '../model/update_check_extension_resp.dart';
import '../model/update_extension_settings.dart';
import 'model/start_config.dart';

abstract class LibgopeedAbi {
  Future<void> init(String cfg);
  Future<String> invoke(String params);
}

abstract class LibgopeedApiSingleton {
  Future<LibgopeedApi> init(StartConfig config);
}

abstract class LibgopeedApi {
  Future<HttpListenResult> startHttp();

  Future<void> stopHttp();

  Future<void> restartHttp();

  Future<Info> info();

  Future<ResolveResult> resolve(Request request);

  Future<String> createTask(CreateTask createTask);

  Future<String> createTaskBatch(CreateTaskBatch createTask);

  Future<void> pauseTask(String id);

  Future<void> pauseTasks(TaskFilter? filter);

  Future<void> continueTask(String id);

  Future<void> continueTasks(TaskFilter? filter);

  Future<void> deleteTask(String id, bool force);

  Future<void> deleteTasks(TaskFilter? filter, bool force);

  Future<Task> getTask(String id);

  Future<List<Task>> getTasks(TaskFilter? filter);

  Future<Map<String, dynamic>> getTaskStats(String id);

  Future<DownloaderConfig> getConfig();

  Future<void> putConfig(DownloaderConfig config);

  Future<void> installExtension(InstallExtension installExtension);

  Future<List<Extension>> getExtensions();

  Future<void> updateExtensionSettings(
      String identity, UpdateExtensionSettings updateExtensionSettings);

  Future<void> switchExtension(
      String identity, SwitchExtension switchExtension);

  Future<void> deleteExtension(String identity);

  Future<UpdateCheckExtensionResp> upgradeCheckExtension(String identity);

  Future<void> upgradeExtension(String identity);

  Future<void> close();

  Future<Response<String>> proxyRequest<T>(String uri,
      {data, Options? options});
}
