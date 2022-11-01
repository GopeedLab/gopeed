import 'dart:convert';
import 'dart:io';

import 'package:dio/adapter.dart';
import 'package:dio/dio.dart';
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/result.dart';
import 'package:gopeed/api/model/task.dart';

import '../core/libgopeed_boot.dart';
import 'model/server_config.dart';

class _Client {
  static _Client? _instance;

  late Dio dio;

  _Client._internal();

  factory _Client() {
    if (_instance == null) {
      _instance = _Client._internal();
      var dio = Dio();
      final isUnixSocket = LibgopeedBoot.instance.config.network == 'unix';
      dio.options.baseUrl = isUnixSocket
          ? 'http://127.0.0.1'
          : 'http://${LibgopeedBoot.instance.config.address}';
      _instance!.dio = dio;
      if (isUnixSocket) {
        (_instance!.dio.httpClientAdapter as DefaultHttpClientAdapter)
            .onHttpClientCreate = (client) {
          client.connectionFactory =
              (Uri uri, String? proxyHost, int? proxyPort) {
            var address = InternetAddress(LibgopeedBoot.instance.config.address,
                type: InternetAddressType.unix);
            return Socket.startConnect(address, 0);
          };
          return client;
        };
      }
    }
    return _instance!;
  }
}

var _client = _Client();

Future<T> _parse<T>(
  Future<Response> Function() fetch,
  T Function(dynamic json)? fromJsonT,
) async {
  try {
    var resp = await fetch();
    if (fromJsonT != null) {
      return Result<T>.fromJson(jsonDecode(resp.data), fromJsonT).data as T;
    } else {
      return null as T;
    }
  } on DioError catch (e) {
    if (e.response == null) {
      throw Exception("Server error");
    }
    throw Exception(
        Result.fromJson(jsonDecode(e.response?.data), (_) => null).msg);
  }
}

Future<Resource> resolve(Request request) async {
  return _parse<Resource>(
      () => _client.dio.post("/api/v1/resolve", data: request),
      (data) => Resource.fromJson(data));
}

Future<String> createTask(CreateTask createTask) async {
  print(jsonEncode(createTask));
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

Future<ServerConfig> getConfig() async {
  return _parse(() => _client.dio.get("/api/v1/config"),
      (data) => ServerConfig.fromJson(data));
}

Future<void> putConfig(ServerConfig config) async {
  return _parse(() => _client.dio.put("/api/v1/config", data: config), null);
}
