import 'dart:io';

import 'package:dart_ipc/dart_ipc.dart';

import '../../util/util.dart';

typedef RpcHandler = void Function(Map<String, dynamic> body);

Future<void> startRpcServer() async {
  var path = Util.isWindows()
      ? r'\\.\pipe\gopeed_host'
      : Util.homePathJoin("gopeed_host.sock");

  final serverSocket = await bind(path);
  final httpServer = HttpServer.listenOn(serverSocket);
  httpServer.forEach((HttpRequest request) async {
    // 读取json请求体
    final content = <int>[];
    await for (var data in request) {
      content.addAll(data);
    }
    final body = String.fromCharCodes(content);
    print('Received request body: $body');
    request.response.close();
  });
}
