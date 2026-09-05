import 'dart:io';

Future<bool> isTcpPortInUse(String host, int port) async {
  try {
    final socket = await Socket.connect(host, port, timeout: const Duration(seconds: 3));
    await socket.close();
    return true;
  } catch (_) {
    return false;
  }
}
