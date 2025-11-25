import 'dart:io';

import 'package:dio/dio.dart';
import 'package:dio/io.dart';
import 'package:flutter/foundation.dart';
import 'package:get/get.dart' as getx;

import '../app/routes/app_pages.dart';
import '../database/database.dart';
import '../util/util.dart';
import 'model/create_task.dart';
import 'model/create_task_batch.dart';
import 'model/downloader_config.dart';
import 'model/extension.dart';
import 'model/install_extension.dart';
import 'model/login.dart';
import 'model/request.dart';
import 'model/resolve_result.dart';
import 'model/result.dart';
import 'model/switch_extension.dart';
import 'model/task.dart';
import 'model/update_check_extension_resp.dart';
import 'model/update_extension_settings.dart';

class _Client {
  static _Client? _instance;

  late Dio dio;

  _Client._internal();

  factory _Client(String network, String address, String apiToken) {
    if (_instance == null) {
      _instance = _Client._internal();
      var dio = Dio();
      final isUnixSocket = network == 'unix';
      var baseUrl = 'http://127.0.0.1/';
      if (!isUnixSocket) {
        if (Util.isWeb()) {
          baseUrl = kDebugMode ? 'http://127.0.0.1:9999/' : '';
        } else {
          baseUrl = 'http://$address/';
        }
      }
      dio.options.baseUrl = baseUrl;
      dio.options.contentType = Headers.jsonContentType;
      dio.options.sendTimeout = const Duration(seconds: 5);
      dio.options.connectTimeout = const Duration(seconds: 5);
      dio.options.receiveTimeout = const Duration(seconds: 60);
      dio.interceptors.add(InterceptorsWrapper(
        onRequest: (options, handler) {
          if (apiToken.isNotEmpty) {
            options.headers['X-Api-Token'] = apiToken;
          }
          if (Util.isWeb()) {
            final token = Database.instance.getWebToken();
            if (token != null) {
              options.headers['Authorization'] = 'Bearer $token';
            }
          }
          handler.next(options);
        },
        onError: (error, handler) {
          // Only web version has a login page
          if (Util.isWeb() && error.response?.statusCode == 401) {
            getx.Get.rootDelegate.offAndToNamed(Routes.LOGIN);
          }
          handler.next(error);
        },
      ));

      _instance!.dio = dio;
      if (isUnixSocket) {
        (_instance!.dio.httpClientAdapter as IOHttpClientAdapter)
            .createHttpClient = () {
          final client = HttpClient();
          client.connectionFactory =
              (Uri uri, String? proxyHost, int? proxyPort) {
            return Socket.startConnect(
                InternetAddress(address, type: InternetAddressType.unix), 0);
          };
          return client;
        };
      }
    }
    return _instance!;
  }
}

class TimeoutException implements Exception {
  final String message;

  TimeoutException(this.message);
}

late _Client _client;

void init(String network, String address, String apiToken) {
  _client = _Client(network, address, apiToken);
}

Future<T> _parse<T>(
  Future<Response> Function() fetch,
  T Function(dynamic json)? fromJsonT,
) async {
  try {
    var resp = await fetch();
    fromJsonT ??= (json) => null as T;
    final result = Result<T>.fromJson(resp.data, fromJsonT);
    if (result.code == 0) {
      return result.data as T;
    } else {
      throw Exception(result);
    }
  } on DioException catch (e) {
    if (e.type == DioExceptionType.sendTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.connectionError) {
      throw TimeoutException("request timeout");
    }
    throw Exception(Result(code: 1000, msg: e.message));
  }
}

Future<ResolveResult> resolve(Request request) async {
  return _parse<ResolveResult>(
      () => _client.dio.post("api/v1/resolve", data: request),
      (data) => ResolveResult.fromJson(data));
}

Future<String> createTask(CreateTask createTask) async {
  return _parse<String>(
      () => _client.dio.post("api/v1/tasks", data: createTask),
      (data) => data as String);
}

Future<List<String>> createTaskBatch(CreateTaskBatch createTaskBatch) async {
  return _parse<List<String>>(
      () => _client.dio.post("api/v1/tasks/batch", data: createTaskBatch),
      (data) => (data as List).map((e) => e as String).toList());
}

Future<List<Task>> getTasks(List<Status> statuses) async {
  return _parse<List<Task>>(
      () => _client.dio.get(
          "/api/v1/tasks?${statuses.map((e) => "status=${e.name}").join("&")}"),
      (data) => (data as List).map((e) => Task.fromJson(e)).toList());
}

Future<void> pauseTask(String id) async {
  return _parse(() => _client.dio.put("api/v1/tasks/$id/pause"), null);
}

Future<void> continueTask(String id) async {
  return _parse(() => _client.dio.put("api/v1/tasks/$id/continue"), null);
}

Future<void> pauseAllTasks(List<String>? ids) async {
  return _parse(
      () => _client.dio.put("api/v1/tasks/pause", queryParameters: {
            "id": ids,
          }),
      null);
}

Future<void> continueAllTasks(List<String>? ids) async {
  return _parse(
      () => _client.dio.put("api/v1/tasks/continue", queryParameters: {
            "id": ids,
          }),
      null);
}

Future<void> deleteTask(String id, bool force) async {
  return _parse(
      () => _client.dio.delete("api/v1/tasks/$id?force=$force"), null);
}

Future<void> deleteTasks(List<String>? ids, bool force) async {
  return _parse(
      () => _client.dio.delete("api/v1/tasks", queryParameters: {
            "id": ids,
            "force": force,
          }),
      null);
}

Future<DownloaderConfig> getConfig() async {
  return _parse(() => _client.dio.get("api/v1/config"),
      (data) => DownloaderConfig.fromJson(data));
}

Future<void> putConfig(DownloaderConfig config) async {
  return _parse(() => _client.dio.put("api/v1/config", data: config), null);
}

Future<void> installExtension(InstallExtension installExtension) async {
  return _parse(
      () => _client.dio.post("api/v1/extensions", data: installExtension),
      null);
}

Future<List<Extension>> getExtensions() async {
  return _parse<List<Extension>>(() => _client.dio.get("api/v1/extensions"),
      (data) => (data as List).map((e) => Extension.fromJson(e)).toList());
}

Future<void> updateExtensionSettings(
    String identity, UpdateExtensionSettings updateExtensionSettings) async {
  return _parse(
      () => _client.dio.put("api/v1/extensions/$identity/settings",
          data: updateExtensionSettings),
      null);
}

Future<void> switchExtension(
    String identity, SwitchExtension switchExtension) async {
  return _parse(
      () => _client.dio
          .put("api/v1/extensions/$identity/switch", data: switchExtension),
      null);
}

Future<void> deleteExtension(String identity) async {
  return _parse(() => _client.dio.delete("api/v1/extensions/$identity"), null);
}

Future<UpdateCheckExtensionResp> upgradeCheckExtension(String identity) async {
  return _parse(() => _client.dio.get("api/v1/extensions/$identity/update"),
      (data) => UpdateCheckExtensionResp.fromJson(data));
}

Future<void> updateExtension(String identity) async {
  return _parse(
      () => _client.dio.post("api/v1/extensions/$identity/update"), null);
}

Future<void> testWebhook(String url) async {
  return _parse(
      () => _client.dio.post("api/v1/webhook/test", data: {"url": url}), null);
}

Future<String> login(LoginReq loginReq) async {
  return _parse(() => _client.dio.post("api/web/login", data: loginReq),
      (data) => data as String);
}

Future<Response<String>> proxyRequest<T>(String uri,
    {data, Options? options}) async {
  options ??= Options();
  options.headers ??= {};
  options.headers!["X-Target-Uri"] = uri;

  // add timestamp to avoid cache
  return _client.dio.request(
      "/api/v1/proxy?t=${DateTime.now().millisecondsSinceEpoch}",
      data: data,
      options: options);
}

String join(String path) {
  final baseUrl = _client.dio.options.baseUrl;
  final cleanBaseUrl = baseUrl.endsWith('/')
      ? baseUrl.substring(0, baseUrl.length - 1)
      : baseUrl;
  return "$cleanBaseUrl/${Util.cleanPath(path)}";
}
