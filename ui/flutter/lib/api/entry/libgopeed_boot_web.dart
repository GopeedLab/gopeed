import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import '../../api/model/create_task.dart';
import '../../api/model/create_task_batch.dart';
import '../../api/model/downloader_config.dart';
import '../../api/model/extension.dart';
import '../../api/model/http_listen_result.dart';
import '../../api/model/info.dart';
import '../../api/model/install_extension.dart';
import '../../api/model/request.dart';
import '../../api/model/resolve_result.dart';
import '../../api/model/result.dart';
import '../../api/model/switch_extension.dart';
import '../../api/model/task.dart';
import '../../api/model/task_filter.dart';
import '../../api/model/update_check_extension_resp.dart';
import '../../api/model/update_extension_settings.dart';
import '../../util/extensions.dart';
import '../../util/util.dart';
import '../api_exception.dart';
import '../libgopeed_boot.dart';
import '../native/libgopeed_interface.dart';
import '../native/model/start_config.dart';
import 'libgopeed_boot_base.dart';

LibgopeedBoot create() => LibgopeedBootWeb();

class LibgopeedBootWeb
    with LibgopeedBootBase
    implements LibgopeedBoot, LibgopeedApi {
  late Dio _dio;
  static const String _apiPrefix = '/api/v1';

  LibgopeedBootWeb();

  Future<T> _fetch<T>(
    Future<Response> Function() fetch,
    T Function(dynamic json)? fromJsonT,
  ) async {
    try {
      var resp = await fetch();
      fromJsonT ??= (json) => null as T;
      final result = Result<T>.fromJson(resp.data, fromJsonT);
      return handleResult(result);
    } on DioException catch (e) {
      if (e.type == DioExceptionType.sendTimeout ||
          e.type == DioExceptionType.receiveTimeout ||
          e.type == DioExceptionType.connectionTimeout) {
        throw ApiException(1000, 'request timeout');
      }
      throw ApiException(1000, e.message ?? "");
    }
  }

  Map<String, dynamic>? _parseTaskFilter(TaskFilter? filter) {
    if (filter == null) {
      return null;
    }

    Map<String, dynamic> params = {};
    if (filter.ids != null) {
      params['ids'] = filter.ids;
    }
    if (filter.statuses != null) {
      params['statuses'] = filter.statuses;
    }
    if (filter.notStatuses != null) {
      params['notStatuses'] = filter.notStatuses;
    }
    if (params.isEmpty) {
      return null;
    }
    return params;
  }

  @override
  Future<LibgopeedApi> init(StartConfig cfg) async {
    _dio = createDio();
    _dio.options.baseUrl = kDebugMode ? 'http://127.0.0.1:9999' : '';
    return this;
  }

  @override
  Future<HttpListenResult> startHttp() async {
    throw UnimplementedError();
  }

  @override
  Future<void> stopHttp() async {
    throw UnimplementedError();
  }

  @override
  Future<void> restartHttp() async {
    throw UnimplementedError();
  }

  @override
  Future<Info> info() {
    return _fetch<Info>(
        () => _dio.get("$_apiPrefix/info"), (data) => Info.fromJson(data));
  }

  @override
  Future<ResolveResult> resolve(Request request) async {
    return _fetch<ResolveResult>(
        () => _dio.post("$_apiPrefix/resolve", data: request),
        (data) => ResolveResult.fromJson(data));
  }

  @override
  Future<String> createTask(CreateTask createTask) async {
    return _fetch<String>(
        () => _dio.post("$_apiPrefix/tasks", data: createTask),
        (data) => data as String);
  }

  @override
  Future<String> createTaskBatch(CreateTaskBatch createTask) async {
    return _fetch<String>(
        () => _dio.post("$_apiPrefix/tasks", data: createTask),
        (data) => data as String);
  }

  @override
  Future<void> pauseTask(String id) async {
    await _fetch<void>(() => _dio.put("$_apiPrefix/tasks/$id/pause"), null);
  }

  @override
  Future<void> pauseTasks(TaskFilter? filter) async {
    await _fetch<void>(
        () => _dio.put("$_apiPrefix/tasks/pause",
            queryParameters: _parseTaskFilter(filter)),
        null);
  }

  @override
  Future<void> continueTask(String id) async {
    await _fetch<void>(() => _dio.put("$_apiPrefix/tasks/$id/continue"), null);
  }

  @override
  Future<void> continueTasks(TaskFilter? filter) async {
    await _fetch<void>(
        () => _dio.put("$_apiPrefix/tasks/continue",
            queryParameters: _parseTaskFilter(filter)),
        null);
  }

  @override
  Future<void> deleteTask(String id, bool force) async {
    await _fetch<void>(
        () => _dio
            .delete("$_apiPrefix/tasks/$id", queryParameters: {'force': force}),
        null);
  }

  @override
  Future<void> deleteTasks(TaskFilter? filter, bool force) async {
    await _fetch<void>(
        () => _dio.delete("$_apiPrefix/tasks",
            queryParameters: _parseTaskFilter(filter)?.apply((it) {
              it['force'] = force;
            })),
        null);
  }

  @override
  Future<Task> getTask(String id) async {
    return _fetch<Task>(
        () => _dio.get("$_apiPrefix/tasks/$id"), (data) => Task.fromJson(data));
  }

  @override
  Future<List<Task>> getTasks(TaskFilter? filter) async {
    return _fetch<List<Task>>(
        () => _dio.get("$_apiPrefix/tasks",
            queryParameters: _parseTaskFilter(filter)),
        (data) => (data as List).map((e) => Task.fromJson(e)).toList());
  }

  @override
  Future<Map<String, dynamic>> getTaskStats(String id) async {
    return _fetch<Map<String, dynamic>>(
        () => _dio.get("$_apiPrefix/tasks/$id/stats"),
        (data) => data as Map<String, dynamic>);
  }

  @override
  Future<DownloaderConfig> getConfig() async {
    return _fetch<DownloaderConfig>(() => _dio.get("$_apiPrefix/config"),
        (data) => DownloaderConfig.fromJson(data));
  }

  @override
  Future<void> putConfig(DownloaderConfig config) async {
    await _fetch<void>(
        () => _dio.put("$_apiPrefix/config", data: config), null);
  }

  @override
  Future<void> installExtension(InstallExtension installExtension) async {
    await _fetch<void>(
        () => _dio.post("$_apiPrefix/extensions", data: installExtension),
        null);
  }

  @override
  Future<List<Extension>> getExtensions() async {
    return _fetch<List<Extension>>(() => _dio.get("$_apiPrefix/extensions"),
        (data) => (data as List).map((e) => Extension.fromJson(e)).toList());
  }

  @override
  Future<void> updateExtensionSettings(
      String identity, UpdateExtensionSettings updateExtensionSettings) async {
    await _fetch<void>(
        () => _dio.put("$_apiPrefix/extensions/$identity/settings",
            data: updateExtensionSettings),
        null);
  }

  @override
  Future<void> switchExtension(
      String identity, SwitchExtension switchExtension) async {
    await _fetch<void>(
        () => _dio.put("$_apiPrefix/extensions/$identity/switch",
            data: switchExtension),
        null);
  }

  @override
  Future<void> deleteExtension(String identity) async {
    await _fetch<void>(
        () => _dio.delete("$_apiPrefix/extensions/$identity"), null);
  }

  @override
  Future<UpdateCheckExtensionResp> upgradeCheckExtension(
      String identity) async {
    return _fetch<UpdateCheckExtensionResp>(
        () => _dio.get("$_apiPrefix/extensions/$identity/upgrade"),
        (data) => UpdateCheckExtensionResp.fromJson(data));
  }

  @override
  Future<void> upgradeExtension(String identity) async {
    await _fetch<void>(
        () => _dio.post("$_apiPrefix/extensions/$identity/upgrade"), null);
  }

  @override
  Future<void> close() async {
    throw UnimplementedError();
  }

  // To avoid CORS, use a backend proxy to forward the request
  @override
  Future<Response<String>> proxyRequest<T>(String uri,
      {data, Options? options}) async {
    options ??= Options();
    options.headers ??= {};
    options.headers!["X-Target-Uri"] = uri;

    // add timestamp to avoid cache
    return _dio.request(
        "/api/v1/proxy?t=${DateTime.now().millisecondsSinceEpoch}",
        data: data,
        options: options);
  }

  String join(String path) {
    return "${_dio.options.baseUrl}/${Util.cleanPath(path)}";
  }
}
