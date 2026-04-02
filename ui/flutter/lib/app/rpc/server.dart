import 'dart:convert';
import 'dart:io';

import 'package:dart_ipc/dart_ipc.dart';

import '../../util/util.dart';

class RpcContext {
  final HttpRequest request;
  final HttpResponse response;
  String? _bodyCache;

  RpcContext(this.request, this.response);

  Future<Map<String, dynamic>> readJSON() async {
    if (_bodyCache == null) {
      final content = <int>[];
      await for (final data in request) {
        content.addAll(data);
      }
      _bodyCache = String.fromCharCodes(content);
    }

    if (_bodyCache!.isEmpty) {
      return {};
    }

    return jsonDecode(_bodyCache!) as Map<String, dynamic>;
  }

  Future<String> readText() async {
    if (_bodyCache == null) {
      final content = <int>[];
      await for (final data in request) {
        content.addAll(data);
      }
      _bodyCache = String.fromCharCodes(content);
    }

    return _bodyCache ?? '';
  }

  Future<void> writeJSON(Map<String, dynamic> data) async {
    response.headers.contentType = ContentType.json;
    response.write(jsonEncode(data));
  }

  Future<void> writeError(String message, [int statusCode = 500]) async {
    response.statusCode = statusCode;
    await writeJSON({'error': message});
  }
}

typedef RouteHandler = Future<void> Function(RpcContext ctx);

class RouteRegistry {
  final Map<String, RouteHandler> _routes = {};

  void register(String path, RouteHandler handler) {
    _routes[path] = handler;
  }

  RouteHandler? getHandler(String path) {
    return _routes[path];
  }
}

class RpcBinding {
  final String network;
  final String address;

  const RpcBinding({
    required this.network,
    required this.address,
  });
}

class RpcServerHandle {
  final RpcBinding binding;
  final HttpServer server;
  final Future<void> Function()? _cleanup;

  RpcServerHandle({
    required this.binding,
    required this.server,
    Future<void> Function()? cleanup,
  }) : _cleanup = cleanup;

  Future<void> close({bool force = true}) async {
    await server.close(force: force);
    if (_cleanup != null) {
      await _cleanup!();
    }
  }
}

Future<RpcBinding> defaultHostRpcBinding() async {
  if (Util.isWindows()) {
    return const RpcBinding(
      network: 'pipe',
      address: r'\\.\pipe\gopeed_host',
    );
  }
  return RpcBinding(
    network: 'unix',
    address: await Util.homePathJoin('gopeed_host.sock'),
  );
}

Future<RpcBinding> defaultWebViewRpcBinding() async {
  if (Util.supportUnixSocket()) {
    return RpcBinding(
      network: 'unix',
      address: await Util.homePathJoin('gopeed_webview.sock'),
    );
  }
  return const RpcBinding(
    network: 'tcp',
    address: '127.0.0.1:0',
  );
}

Future<RpcServerHandle> startRpcServer({
  RpcBinding? binding,
  Map<String, RouteHandler>? routes,
}) async {
  final rpcBinding = binding ?? await defaultHostRpcBinding();
  Future<void> Function()? cleanup;
  late final HttpServer httpServer;

  switch (rpcBinding.network) {
    case 'pipe':
      final serverSocket = await bind(rpcBinding.address);
      httpServer = HttpServer.listenOn(serverSocket);
      break;
    case 'unix':
      final socketFile = File(rpcBinding.address);
      if (await socketFile.exists()) {
        try {
          await socketFile.delete();
        } catch (_) {}
      }
      final serverSocket = await bind(rpcBinding.address);
      httpServer = HttpServer.listenOn(serverSocket);
      cleanup = () async {
        if (await socketFile.exists()) {
          try {
            await socketFile.delete();
          } catch (_) {}
        }
      };
      break;
    case 'tcp':
      final separator = rpcBinding.address.lastIndexOf(':');
      if (separator <= 0 || separator == rpcBinding.address.length - 1) {
        throw ArgumentError('invalid tcp rpc address: ${rpcBinding.address}');
      }
      final host = rpcBinding.address.substring(0, separator);
      final port = int.parse(rpcBinding.address.substring(separator + 1));
      httpServer = await HttpServer.bind(host, port);
      break;
    default:
      throw ArgumentError('unsupported rpc network: ${rpcBinding.network}');
  }

  final registry = RouteRegistry();
  routes?.forEach(registry.register);

  httpServer.forEach((request) async {
    final ctx = RpcContext(request, request.response);
    try {
      final handler = registry.getHandler(request.uri.path);
      if (handler != null) {
        await handler(ctx);
      } else {
        await ctx.writeError('Route not found: ${request.uri.path}', 404);
      }
    } catch (e) {
      await ctx.writeError('Internal server error: $e', 500);
    } finally {
      await request.response.close();
    }
  });

  var resolvedBinding = rpcBinding;
  if (rpcBinding.network == 'tcp') {
    resolvedBinding = RpcBinding(
      network: rpcBinding.network,
      address: '${httpServer.address.address}:${httpServer.port}',
    );
  }

  return RpcServerHandle(
    binding: resolvedBinding,
    server: httpServer,
    cleanup: cleanup,
  );
}
