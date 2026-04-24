import 'dart:convert';

import 'package:dio/dio.dart';

import '../../api/model/create_task.dart';
import '../../api/model/result.dart';
import 'server.dart';

typedef HostCreateHandler = Future<void> Function(
  CreateTask createTask,
  bool silent,
);

typedef HostForwardHandler = Future<dynamic> Function({
  required String path,
  required String method,
  dynamic data,
  Map<String, dynamic>? query,
});

class HostRpcService {
  HostRpcService._();

  static final HostRpcService instance = HostRpcService._();

  RpcServerHandle? _server;

  Future<void> start({
    required HostCreateHandler onCreate,
    required HostForwardHandler onForward,
  }) async {
    if (_server != null) {
      return;
    }
    _server = await startRpcServer(
      routes: {
        '/create': (ctx) async {
          final meta =
              ctx.request.headers['X-Gopeed-Host-Meta']?.firstOrNull ?? '{}';
          final jsonMeta = jsonDecode(meta);
          final silent = jsonMeta['silent'] as bool? ?? false;
          final params = await ctx.readText();
          final createTask = CreateTask.fromJson(_decodeParams(params));
          await onCreate(createTask, silent);
        },
        '/forward': (ctx) async {
          try {
            final body = await ctx.readJSON();
            final method = (body['method'] as String?)?.toUpperCase() ?? 'GET';
            final path = (body['path'] as String?) ?? '/';
            final data = body['data'];
            final query = body['query'] as Map<String, dynamic>?;
            final response = await onForward(
              path: path,
              method: method,
              data: data,
              query: query,
            );
            if (response is Map<String, dynamic>) {
              await ctx.writeJSON(response);
            } else {
              await ctx.writeJSON((response as dynamic).data);
            }
          } catch (e) {
            if (e is DioException && e.response != null) {
              await ctx.writeJSON(e.response!.data);
            } else {
              await ctx.writeJSON(Result(code: 1, msg: e.toString()).toJson());
            }
          }
        },
      },
    );
  }

  Future<void> stop() async {
    final server = _server;
    _server = null;
    if (server != null) {
      await server.close();
    }
  }

  Map<String, dynamic> _decodeParams(String params) {
    final safeParams = params.replaceAll('"', '').replaceAll(' ', '+');
    final paramsJson =
        String.fromCharCodes(base64Decode(base64.normalize(safeParams)));
    return jsonDecode(paramsJson);
  }
}
