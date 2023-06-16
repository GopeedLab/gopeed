import 'dart:io';

import 'package:dio/dio.dart';
import 'package:dio/io.dart';
import 'package:flutter/foundation.dart';
import 'package:gopeed/api/model/create_task_batch.dart';
import 'model/resolve_result.dart';
import '../util/util.dart';
import 'model/create_task.dart';
import 'model/request.dart';
import 'model/result.dart';
import 'model/task.dart';

import 'model/downloader_config.dart';

class _Client {
  static _Client? _instance;

  late Dio dio;

  _Client._internal();

  factory _Client(String network, String address, String apiToken) {
    if (_instance == null) {
      _instance = _Client._internal();
      var dio = Dio();
      final isUnixSocket = network == 'unix';
      var baseUrl = 'http://127.0.0.1';
      if (!isUnixSocket) {
        if (Util.isWeb()) {
          baseUrl = kDebugMode ? 'http://127.0.0.1:9999' : '';
        } else {
          baseUrl = 'http://$address';
        }
      }
      dio.options.baseUrl = baseUrl;
      dio.options.contentType = Headers.jsonContentType;
      dio.options.connectTimeout = const Duration(seconds: 5);
      dio.options.receiveTimeout = const Duration(seconds: 60);
      dio.interceptors.add(InterceptorsWrapper(onRequest: (options, handler) {
        if (apiToken.isNotEmpty) {
          options.headers['X-Api-Token'] = apiToken;
        }
        handler.next(options);
      }));

      _instance!.dio = dio;
      if (isUnixSocket) {
        (_instance!.dio.httpClientAdapter as IOHttpClientAdapter)
            .onHttpClientCreate = (client) {
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
  } on DioError catch (e) {
    throw Exception(Result(code: 1000, msg: e.message));
  }
}

Future<ResolveResult> resolve(Request request) async {
  return _parse<ResolveResult>(
      () => _client.dio.post("/api/v1/resolve", data: request),
      (data) => ResolveResult.fromJson(data));
}

Future<String> createTask(CreateTask createTask) async {
  return _parse<String>(
      () => _client.dio.post("/api/v1/tasks", data: createTask),
      (data) => data as String);
}

Future<List<String>> createTaskBatch(CreateTaskBatch createTaskBatch) async {
  return _parse<List<String>>(
      () => _client.dio.post("/api/v1/tasks/batch", data: createTaskBatch),
      (data) => (data as List).map((e) => e as String).toList());
}

Future<List<Task>> getTasks(List<Status> statuses) async {
  return _parse<List<Task>>(
      () => _client.dio
          .get("/api/v1/tasks?status=${statuses.map((e) => e.name).join(",")}"),
      (data) => (data as List).map((e) => Task.fromJson(e)).toList());
}

Future<void> pauseTask(String id) async {
  return _parse(() => _client.dio.put("/api/v1/tasks/$id/pause"), null);
}

Future<void> continueTask(String id) async {
  return _parse(() => _client.dio.put("/api/v1/tasks/$id/continue"), null);
}

Future<void> pauseAllTasks() async {
  return _parse(() => _client.dio.put("/api/v1/tasks/pause"), null);
}

Future<void> continueAllTasks() async {
  return _parse(() => _client.dio.put("/api/v1/tasks/continue"), null);
}

Future<void> deleteTask(String id, bool force) async {
  return _parse(
      () => _client.dio.delete("/api/v1/tasks/$id?force=$force"), null);
}

Future<DownloaderConfig> getConfig() async {
  return _parse(() => _client.dio.get("/api/v1/config"),
      (data) => DownloaderConfig.fromJson(data));
}

Future<void> putConfig(DownloaderConfig config) async {
  return _parse(() => _client.dio.put("/api/v1/config", data: config), null);
}

Future<Response<String>> proxyRequest<T>(String uri,
    {data, Options? options}) async {
  options ??= Options();
  options.headers ??= {};
  options.headers!["X-Target-Uri"] = uri;

  // add timestamp to avoid cache
  return _client.dio.request("/api/v1/proxy?t=${DateTime.now()}",
      data: data, options: options);
}
