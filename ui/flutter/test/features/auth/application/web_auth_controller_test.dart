import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';
import 'package:gopeed/features/auth/application/web_auth_controller.dart';

void main() {
  test('successful login relies on the server cookie and unauthorized requires login', () async {
    api.setTransportForTesting(_LoginTransport());
    final controller = WebAuthController();
    addTearDown(controller.dispose);

    final failure = await controller.login(username: 'gopeed', password: 'secret');

    expect(failure, isNull);
    controller.requireLogin();
    expect(controller.loginRequired, isTrue);
  });
}

class _LoginTransport implements GopeedTransport {
  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    expect(path, 'api/web/login');
    expect(method, 'POST');
    return {'code': 0};
  }

  @override
  Future<Response<String>> proxyRequest(String uri, {dynamic data, Options? options}) {
    throw UnimplementedError();
  }

  @override
  String join(String path) => path;
}
