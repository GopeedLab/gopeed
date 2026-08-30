import 'package:dio/dio.dart';

import '../core/network/gopeed/gopeed_transport.dart';

import 'model/create_task.dart';
import 'model/create_task_batch.dart';
import 'model/downloader_config.dart';
import 'model/extension.dart';
import 'model/install_extension.dart';
import 'model/login.dart';
import 'model/resolve_result.dart';
import 'model/resolve_task.dart';
import 'model/result.dart';
import 'model/switch_extension.dart';
import 'model/task.dart';
import 'model/update_check_extension_resp.dart';
import 'model/update_extension_settings.dart';

class ApiTimeoutException implements Exception {
  const ApiTimeoutException(this.message);

  final String message;

  @override
  String toString() => message;
}

GopeedTransport? _transport;

void setTransportForTesting(GopeedTransport transport) {
  _transport = transport;
}

void init(String network, String address, String apiToken, {void Function()? onUnauthorized}) {
  _transport = createGopeedTransport(
    GopeedTransportConfig(network: network, address: address, apiToken: apiToken, onUnauthorized: onUnauthorized),
  );
}

void initDefault({String address = '127.0.0.1:9999', String apiToken = ''}) {
  if (_transport == null) {
    init('tcp', address, apiToken);
  }
}

Future<dynamic> _request(String path, {String method = 'GET', dynamic data, Map<String, dynamic>? queryParameters}) {
  initDefault();
  return _transport!.request(path, method: method, data: data, queryParameters: queryParameters);
}

Future<T> _parse<T>(Future<dynamic> Function() fetch, T Function(dynamic json)? fromJsonT) async {
  initDefault();
  try {
    final resp = await fetch();
    fromJsonT ??= (json) => null as T;
    final result = Result<T>.fromJson(resp as Map<String, dynamic>, fromJsonT);
    if (result.code == 0) {
      return result.data as T;
    }
    final message = result.msg;
    throw Exception(message == null || message.isEmpty ? 'API error ${result.code}' : message);
  } on DioException catch (e) {
    if (e.type == DioExceptionType.sendTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.connectionError) {
      throw const ApiTimeoutException('request timeout');
    }
    throw Exception(e.message ?? 'request failed');
  }
}

Future<ResolveResult> resolve(ResolveTask resolveTask) {
  if (resolveTask.req == null) {
    throw ArgumentError('resolve request is required');
  }
  return _parse<ResolveResult>(
    () => _request('api/v1/resolve', method: 'POST', data: resolveTask),
    (data) => ResolveResult.fromJson(data as Map<String, dynamic>),
  );
}

Future<String> createTask(CreateTask createTask) {
  return _parse<String>(() => _request('api/v1/tasks', method: 'POST', data: createTask), (data) => data as String);
}

Future<List<String>> createTaskBatch(CreateTaskBatch createTaskBatch) {
  return _parse<List<String>>(
    () => _request('api/v1/tasks/batch', method: 'POST', data: createTaskBatch),
    (data) => (data as List<dynamic>).map((e) => e as String).toList(),
  );
}

Future<void> patchTask(String id, ResolveTask patchTask) {
  return _parse<void>(() => _request('api/v1/tasks/$id', method: 'PATCH', data: patchTask), null);
}

Future<List<Task>> getTasks(List<Status> statuses) {
  final query = statuses.map((e) => 'status=${e.name}').join('&');
  return _parse<List<Task>>(
    () => _request('/api/v1/tasks?$query'),
    (data) => (data as List<dynamic>).map((e) => Task.fromJson(e as Map<String, dynamic>)).toList(),
  );
}

Future<TaskRuntimeStatus> getTaskStatus(String id) {
  return _parse<TaskRuntimeStatus>(
    () => _request('/api/v1/tasks/$id/status'),
    (data) => TaskRuntimeStatus.fromJson(data as Map<String, dynamic>),
  );
}

Future<Map<String, dynamic>> getTaskStats(String id) {
  return _parse<Map<String, dynamic>>(
    () => _request('/api/v1/tasks/$id/stats'),
    (data) =>
        data is Map ? {for (final entry in data.entries) entry.key.toString(): entry.value} : const <String, dynamic>{},
  );
}

Future<void> pauseTask(String id) {
  return _parse<void>(() => _request('api/v1/tasks/$id/pause', method: 'PUT'), null);
}

Future<void> continueTask(String id) {
  return _parse<void>(() => _request('api/v1/tasks/$id/continue', method: 'PUT'), null);
}

Future<void> pauseAllTasks(List<String>? ids) {
  return _parse<void>(() => _request('api/v1/tasks/pause', method: 'PUT', queryParameters: {'id': ids}), null);
}

Future<void> continueAllTasks(List<String>? ids) {
  return _parse<void>(() => _request('api/v1/tasks/continue', method: 'PUT', queryParameters: {'id': ids}), null);
}

Future<void> deleteTask(String id, bool force) {
  return _parse<void>(() => _request('api/v1/tasks/$id', method: 'DELETE', queryParameters: {'force': force}), null);
}

Future<void> deleteTasks(List<String>? ids, bool force) {
  return _parse<void>(
    () => _request('api/v1/tasks', method: 'DELETE', queryParameters: {'id': ids, 'force': force}),
    null,
  );
}

Future<DownloaderConfig> getConfig() {
  return _parse<DownloaderConfig>(
    () => _request('api/v1/config'),
    (data) => DownloaderConfig.fromJson(data as Map<String, dynamic>),
  );
}

Future<void> putConfig(DownloaderConfig config) {
  return _parse<void>(() => _request('api/v1/config', method: 'PUT', data: config.toJson()), null);
}

Future<String> installExtension(InstallExtension installExtension) {
  return _parse<String>(
    () => _request('api/v1/extensions', method: 'POST', data: installExtension.toJson()),
    (data) => data as String,
  );
}

Future<List<Extension>> getExtensions() {
  return _parse<List<Extension>>(
    () => _request('api/v1/extensions'),
    (data) => (data as List<dynamic>).map((e) => Extension.fromJson(e as Map<String, dynamic>)).toList(),
  );
}

Future<void> updateExtensionSettings(String identity, UpdateExtensionSettings updateExtensionSettings) {
  return _parse<void>(
    () => _request('api/v1/extensions/$identity/settings', method: 'PUT', data: updateExtensionSettings.toJson()),
    null,
  );
}

Future<void> switchExtension(String identity, SwitchExtension switchExtension) {
  return _parse<void>(
    () => _request('api/v1/extensions/$identity/switch', method: 'PUT', data: switchExtension.toJson()),
    null,
  );
}

Future<void> deleteExtension(String identity) {
  return _parse<void>(() => _request('api/v1/extensions/$identity', method: 'DELETE'), null);
}

Future<UpdateCheckExtensionResp> upgradeCheckExtension(String identity) {
  return _parse<UpdateCheckExtensionResp>(
    () => _request('api/v1/extensions/$identity/update'),
    (data) => UpdateCheckExtensionResp.fromJson(data as Map<String, dynamic>),
  );
}

Future<void> updateExtension(String identity) {
  return _parse<void>(() => _request('api/v1/extensions/$identity/update', method: 'POST'), null);
}

Future<void> testWebhook(String url) {
  return _parse<void>(() => _request('api/v1/webhook/test', method: 'POST', data: {'url': url}), null);
}

Future<void> login(LoginReq loginReq) {
  return _parse<void>(() => _request('api/web/login', method: 'POST', data: loginReq.toJson()), null);
}

Future<Response<String>> proxyRequest<T>(String uri, {dynamic data, Options? options}) async {
  initDefault();
  return _transport!.proxyRequest(uri, data: data, options: options);
}

String join(String path) {
  initDefault();
  return _transport!.join(path);
}

Future<dynamic> forward(
  String path, {
  String method = 'GET',
  dynamic data,
  Map<String, dynamic>? queryParameters,
}) async {
  return _request(path, method: method, data: data, queryParameters: queryParameters);
}
