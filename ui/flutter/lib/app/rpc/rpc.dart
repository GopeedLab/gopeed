import 'dart:io';
import 'dart:convert';

import 'package:dart_ipc/dart_ipc.dart';
import 'package:path_provider/path_provider.dart';

import '../../util/util.dart';

/// RPC context class that contains request information and response methods
class RpcContext {
  final HttpRequest request;
  final HttpResponse response;
  String? _bodyCache;

  RpcContext(this.request, this.response);

  /// Read and parse JSON request body
  Future<Map<String, dynamic>> readJSON() async {
    if (_bodyCache == null) {
      final content = <int>[];
      await for (var data in request) {
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
      await for (var data in request) {
        content.addAll(data);
      }
      _bodyCache = String.fromCharCodes(content);
    }

    return _bodyCache ?? '';
  }

  /// Write JSON response
  Future<void> writeJSON(Map<String, dynamic> data) async {
    response.headers.contentType = ContentType.json;
    response.write(jsonEncode(data));
    await response.close();
  }

  /// Write error response
  Future<void> writeError(String message, [int statusCode = 500]) async {
    response.statusCode = statusCode;
    await writeJSON({'error': message});
  }
}

/// Route handler type definition
typedef RouteHandler = Future<void> Function(RpcContext ctx);

/// Route registry
class RouteRegistry {
  final Map<String, RouteHandler> _routes = {};

  /// Register a route
  void register(String path, RouteHandler handler) {
    _routes[path] = handler;
  }

  /// Get route handler
  RouteHandler? getHandler(String path) {
    return _routes[path];
  }

  /// Get all registered routes
  List<String> get routes => _routes.keys.toList();
}

/// Start RPC server
Future<void> startRpcServer([Map<String, RouteHandler>? routes]) async {
  String path;
  if (Util.isWindows()) {
    path = r'\\.\pipe\gopeed_host';
  } else {
    path = await Util.homePathJoin("gopeed_host.sock");
    // try to delete existing socket file
    final socketFile = File(path);
    if (await socketFile.exists()) {
      try {
        await socketFile.delete();
      } catch (e) {
        // ignore
      }
    }
  }

  // Create route registry
  final registry = RouteRegistry();

  // Register provided routes
  if (routes != null) {
    routes.forEach((path, handler) {
      registry.register(path, handler);
    });
  }

  final serverSocket = await bind(path);
  final httpServer = HttpServer.listenOn(serverSocket);

  httpServer.forEach((HttpRequest request) async {
    final ctx = RpcContext(request, request.response);

    try {
      // Get request path
      final requestPath = request.uri.path;
      // Find route handler
      final handler = registry.getHandler(requestPath);
      if (handler != null) {
        // Execute route handler
        await handler(ctx);
      } else {
        // 404 Route not found
        await ctx.writeError('Route not found: $requestPath', 404);
      }
    } catch (e) {
      await ctx.writeError('Internal server error: $e', 500);
    } finally {
      await request.response.close();
    }
  });
}
