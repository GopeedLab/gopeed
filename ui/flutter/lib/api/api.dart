import 'dart:convert';
import 'dart:io';

import 'package:dio/adapter.dart';
import 'package:dio/dio.dart';
import '../util/util.dart';
import 'model/create_task.dart';
import 'model/request.dart';
import 'model/resource.dart';
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
      dio.options.baseUrl = isUnixSocket
          ? 'http://127.0.0.1'
          : (Util.isWeb() ? "" : 'http://$address');
      dio.interceptors.add(InterceptorsWrapper(onRequest: (options, handler) {
        if (apiToken.isNotEmpty) {
          options.headers['Authorization'] = apiToken;
        }
        handler.next(options);
      }));

      _instance!.dio = dio;
      if (isUnixSocket) {
        (_instance!.dio.httpClientAdapter as DefaultHttpClientAdapter)
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
    if (fromJsonT != null) {
      return Result<T>.fromJson(resp.data, fromJsonT).data as T;
    } else {
      return null as T;
    }
  } on DioError catch (e) {
    if (e.response == null) {
      throw Exception("Server error");
    }
    throw Exception(Result.fromJson(e.response?.data, (_) => null).msg);
  }
}

Future<Resource> resolve(Request request) async {
  return _parse<Resource>(
      () => _client.dio.post("/api/v1/resolve", data: request),
      (data) => Resource.fromJson(data));
}

Future<String> createTask(CreateTask createTask) async {
  return _parse<String>(
      () => _client.dio.post("/api/v1/tasks", data: createTask),
      (data) => data as String);
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
