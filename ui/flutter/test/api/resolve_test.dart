import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resolve_task.dart';

void main() {
  test('resolve sends the complete resolve task for magnet links', () async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    addTearDown(server.close);

    late Map<String, dynamic> requestBody;
    final responseHandled = Completer<void>();
    server.listen((request) async {
      requestBody = jsonDecode(await utf8.decoder.bind(request).join()) as Map<String, dynamic>;
      request.response.headers.contentType = ContentType.json;
      request.response.write(
        jsonEncode({
          'code': 0,
          'data': {
            'id': 'resolved-id',
            'res': {'name': 'sample', 'size': 0, 'range': false, 'files': <Object>[], 'hash': ''},
          },
        }),
      );
      await request.response.close();
      responseHandled.complete();
    });

    api.init('tcp', '127.0.0.1:${server.port}', '');
    final result = await api.resolve(
      ResolveTask(
        req: Request(url: 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567'),
        opts: Options(path: '/downloads', selectFiles: const []),
      ),
    );
    await responseHandled.future;

    expect(result.id, 'resolved-id');
    expect(requestBody.keys, containsAll(<String>['req', 'opts']));
    expect(
      (requestBody['req'] as Map<String, dynamic>)['url'],
      'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
    );
    expect((requestBody['opts'] as Map<String, dynamic>)['path'], '/downloads');
  });
}
