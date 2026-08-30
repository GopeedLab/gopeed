import 'dart:io';

import 'package:dio/dio.dart';
import 'package:dio/io.dart';
import 'package:flutter/foundation.dart';

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

class _Client {
  _Client._();

  static final _Client instance = _Client._();

  late Dio dio;
  bool initialized = false;

  void init(
    String network,
    String address,
    String apiToken, {
    String? Function()? webTokenProvider,
    VoidCallback? onUnauthorized,
  }) {
    final isUnixSocket = network == 'unix';
    final baseUrl = isUnixSocket
        ? 'http://127.0.0.1/'
        : kIsWeb
        ? kDebugMode
              ? 'http://127.0.0.1:9999/'
              : ''
        : 'http://$address/';
    dio = Dio(
      BaseOptions(
        baseUrl: baseUrl,
        contentType: Headers.jsonContentType,
        sendTimeout: const Duration(seconds: 5),
        connectTimeout: const Duration(seconds: 5),
        receiveTimeout: const Duration(seconds: 120),
      ),
    );
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (apiToken.isNotEmpty) {
            options.headers['X-Api-Token'] = apiToken;
          }
          if (kIsWeb && options.path != 'api/web/login') {
            final token = webTokenProvider?.call();
            if (token != null && token.isNotEmpty) {
              options.headers['Authorization'] = 'Bearer $token';
            }
          }
          handler.next(options);
        },
        onError: (error, handler) {
          if (kIsWeb && error.response?.statusCode == 401) {
            onUnauthorized?.call();
          }
          handler.next(error);
        },
      ),
    );
    if (isUnixSocket) {
      (dio.httpClientAdapter as IOHttpClientAdapter).createHttpClient = () {
        final client = HttpClient();
        client.connectionFactory = (Uri uri, String? proxyHost, int? proxyPort) {
          return Socket.startConnect(InternetAddress(address, type: InternetAddressType.unix), 0);
        };
        return client;
      };
    }
    initialized = true;
  }
}

void init(
  String network,
  String address,
  String apiToken, {
  String? Function()? webTokenProvider,
  VoidCallback? onUnauthorized,
}) {
  _Client.instance.init(network, address, apiToken, webTokenProvider: webTokenProvider, onUnauthorized: onUnauthorized);
}

void initDefault({String address = '127.0.0.1:9999', String apiToken = ''}) {
  if (!_Client.instance.initialized) {
    init('tcp', address, apiToken);
  }
}

Future<T> _parse<T>(Future<Response<dynamic>> Function() fetch, T Function(dynamic json)? fromJsonT) async {
  initDefault();
  try {
    final resp = await fetch();
    fromJsonT ??= (json) => null as T;
    final result = Result<T>.fromJson(resp.data, fromJsonT);
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
    () => _Client.instance.dio.post('api/v1/resolve', data: resolveTask),
    (data) => ResolveResult.fromJson(data as Map<String, dynamic>),
  );
}

Future<String> createTask(CreateTask createTask) {
  return _parse<String>(() => _Client.instance.dio.post('api/v1/tasks', data: createTask), (data) => data as String);
}

Future<List<String>> createTaskBatch(CreateTaskBatch createTaskBatch) {
  return _parse<List<String>>(
    () => _Client.instance.dio.post('api/v1/tasks/batch', data: createTaskBatch),
    (data) => (data as List<dynamic>).map((e) => e as String).toList(),
  );
}

Future<void> patchTask(String id, ResolveTask patchTask) {
  return _parse<void>(() => _Client.instance.dio.patch('api/v1/tasks/$id', data: patchTask), null);
}

Future<List<Task>> getTasks(List<Status> statuses) {
  final query = statuses.map((e) => 'status=${e.name}').join('&');
  return _parse<List<Task>>(
    () => _Client.instance.dio.get('/api/v1/tasks?$query'),
    (data) => (data as List<dynamic>).map((e) => Task.fromJson(e as Map<String, dynamic>)).toList(),
  );
}

Future<TaskRuntimeStatus> getTaskStatus(String id) {
  return _parse<TaskRuntimeStatus>(
    () => _Client.instance.dio.get('/api/v1/tasks/$id/status'),
    (data) => TaskRuntimeStatus.fromJson(data as Map<String, dynamic>),
  );
}

Future<Map<String, dynamic>> getTaskStats(String id) {
  return _parse<Map<String, dynamic>>(
    () => _Client.instance.dio.get('/api/v1/tasks/$id/stats'),
    (data) =>
        data is Map ? {for (final entry in data.entries) entry.key.toString(): entry.value} : const <String, dynamic>{},
  );
}

Future<void> pauseTask(String id) {
  return _parse<void>(() => _Client.instance.dio.put('api/v1/tasks/$id/pause'), null);
}

Future<void> continueTask(String id) {
  return _parse<void>(() => _Client.instance.dio.put('api/v1/tasks/$id/continue'), null);
}

Future<void> pauseAllTasks(List<String>? ids) {
  return _parse<void>(() => _Client.instance.dio.put('api/v1/tasks/pause', queryParameters: {'id': ids}), null);
}

Future<void> continueAllTasks(List<String>? ids) {
  return _parse<void>(() => _Client.instance.dio.put('api/v1/tasks/continue', queryParameters: {'id': ids}), null);
}

Future<void> deleteTask(String id, bool force) {
  return _parse<void>(() => _Client.instance.dio.delete('api/v1/tasks/$id?force=$force'), null);
}

Future<void> deleteTasks(List<String>? ids, bool force) {
  return _parse<void>(
    () => _Client.instance.dio.delete('api/v1/tasks', queryParameters: {'id': ids, 'force': force}),
    null,
  );
}

Future<DownloaderConfig> getConfig() {
  return _parse<DownloaderConfig>(
    () => _Client.instance.dio.get('api/v1/config'),
    (data) => DownloaderConfig.fromJson(data as Map<String, dynamic>),
  );
}

Future<void> putConfig(DownloaderConfig config) {
  return _parse<void>(() => _Client.instance.dio.put('api/v1/config', data: config.toJson()), null);
}

Future<String> installExtension(InstallExtension installExtension) {
  return _parse<String>(
    () => _Client.instance.dio.post('api/v1/extensions', data: installExtension.toJson()),
    (data) => data as String,
  );
}

Future<List<Extension>> getExtensions() {
  return _parse<List<Extension>>(
    () => _Client.instance.dio.get('api/v1/extensions'),
    (data) => (data as List<dynamic>).map((e) => Extension.fromJson(e as Map<String, dynamic>)).toList(),
  );
}

Future<void> updateExtensionSettings(String identity, UpdateExtensionSettings updateExtensionSettings) {
  return _parse<void>(
    () => _Client.instance.dio.put('api/v1/extensions/$identity/settings', data: updateExtensionSettings.toJson()),
    null,
  );
}

Future<void> switchExtension(String identity, SwitchExtension switchExtension) {
  return _parse<void>(
    () => _Client.instance.dio.put('api/v1/extensions/$identity/switch', data: switchExtension.toJson()),
    null,
  );
}

Future<void> deleteExtension(String identity) {
  return _parse<void>(() => _Client.instance.dio.delete('api/v1/extensions/$identity'), null);
}

Future<UpdateCheckExtensionResp> upgradeCheckExtension(String identity) {
  return _parse<UpdateCheckExtensionResp>(
    () => _Client.instance.dio.get('api/v1/extensions/$identity/update'),
    (data) => UpdateCheckExtensionResp.fromJson(data as Map<String, dynamic>),
  );
}

Future<void> updateExtension(String identity) {
  return _parse<void>(() => _Client.instance.dio.post('api/v1/extensions/$identity/update'), null);
}

Future<void> testWebhook(String url) {
  return _parse<void>(() => _Client.instance.dio.post('api/v1/webhook/test', data: {'url': url}), null);
}

Future<String> login(LoginReq loginReq) {
  return _parse<String>(
    () => _Client.instance.dio.post('api/web/login', data: loginReq.toJson()),
    (data) => data as String,
  );
}

Future<Response<String>> proxyRequest<T>(String uri, {dynamic data, Options? options}) async {
  initDefault();
  options ??= Options();
  options.headers ??= {};
  options.headers!['X-Target-Uri'] = uri;
  return _Client.instance.dio.request<String>(
    '/api/v1/proxy?t=${DateTime.now().millisecondsSinceEpoch}',
    data: data,
    options: options,
  );
}

String join(String path) {
  initDefault();
  final baseUrl = _Client.instance.dio.options.baseUrl;
  final cleanBaseUrl = baseUrl.endsWith('/') ? baseUrl.substring(0, baseUrl.length - 1) : baseUrl;
  final cleanPath = path.startsWith('/') ? path.substring(1) : path;
  return '$cleanBaseUrl/$cleanPath';
}

Future<Response<dynamic>> forward(
  String path, {
  String method = 'GET',
  dynamic data,
  Map<String, dynamic>? queryParameters,
}) async {
  initDefault();
  return _Client.instance.dio.request<dynamic>(
    path,
    data: data,
    queryParameters: queryParameters,
    options: Options(method: method),
  );
}
