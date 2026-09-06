import 'dart:io';

Future<ServerSocket> bindIpc(String path) async {
  throw UnsupportedError('IPC sockets are unavailable on the web');
}
