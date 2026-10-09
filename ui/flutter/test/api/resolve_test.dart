import 'dart:convert';

import 'package:dio/dio.dart' as dio;
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/patch_task.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resolve_task.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  test('resolve omits download options and creation submits them separately', () async {
    final transport = _RecordingTransport();
    api.setTransportForTesting(transport);
    final result = await api.resolve(
      ResolveTask(req: Request(url: 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567')),
    );

    expect(result.id, 'resolved-id');
    final requestBody = transport.data as Map<String, dynamic>;
    expect(requestBody.keys, equals(<String>['req']));
    expect(
      (requestBody['req'] as Map<String, dynamic>)['url'],
      'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
    );
    expect(requestBody, isNot(contains('opts')));

    final id = await api.createTask(
      CreateTask(
        rid: result.id,
        opts: Options(path: '/chosen-after-resolve', selectFiles: [1]),
      ),
    );
    expect(id, 'created-id');
    final createBody = transport.data as Map<String, dynamic>;
    expect(createBody['rid'], 'resolved-id');
    expect((createBody['opts'] as Map<String, dynamic>)['path'], '/chosen-after-resolve');
    expect((createBody['opts'] as Map<String, dynamic>)['selectFiles'], [1]);
  });

  test('patch preserves options in its independent request model', () async {
    final transport = _RecordingTransport();
    api.setTransportForTesting(transport);
    await api.patchTask('task-id', PatchTask(opts: Options(selectFiles: [1])));
    final requestBody = transport.data as Map<String, dynamic>;
    expect((requestBody['opts'] as Map<String, dynamic>)['selectFiles'], [1]);
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
    if (path == 'api/v1/tasks') {
      return {'code': 0, 'data': 'created-id'};
    }
    if (method == 'PATCH') {
      return {'code': 0, 'data': null};
    }
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
