import 'dart:convert';

import 'package:dio/dio.dart' as dio;
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resolve_task.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  test('resolve sends the complete resolve task for magnet links', () async {
    final transport = _RecordingTransport();
    api.setTransportForTesting(transport);
    final result = await api.resolve(
      ResolveTask(
        req: Request(url: 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567'),
        opts: Options(path: '/downloads', selectFiles: const []),
      ),
    );

    expect(result.id, 'resolved-id');
    final requestBody = transport.data as Map<String, dynamic>;
    expect(requestBody.keys, containsAll(<String>['req', 'opts']));
    expect(
      (requestBody['req'] as Map<String, dynamic>)['url'],
      'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
    );
    expect((requestBody['opts'] as Map<String, dynamic>)['path'], '/downloads');
  });
}

class _RecordingTransport implements GopeedTransport {
  dynamic data;

  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    this.data = jsonDecode(jsonEncode(data));
    return {
      'code': 0,
      'data': {
        'id': 'resolved-id',
        'res': {'name': 'sample', 'size': 0, 'range': false, 'files': <Object>[], 'hash': ''},
      },
    };
  }

  @override
  String join(String path) => path;

  @override
  Future<dio.Response<String>> proxyRequest(String uri, {dynamic data, dio.Options? options}) {
    throw UnimplementedError();
  }
}
