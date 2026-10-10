import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/gopeed_site_api.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  test('summary requests are shared and cached; explicit refresh requests fresh data', () async {
    final pending = Completer<Response<String>>();
    final transport = _Transport()..onProxy = (_) => pending.future;
    api.setTransportForTesting(transport);
    final site = GopeedSiteApi.instance;
    final first = site.getExtensions(query: 'cache-test', forceRefresh: true);
    final second = site.getExtensions(query: 'cache-test');
    expect(transport.urls.length, 1);
    expect(Uri.parse(transport.urls.single).queryParameters['view'], 'summary');
    pending.complete(
      _response({
        'data': [
          {'id': 'test'},
        ],
      }),
    );
    expect((await first).data.single.hasDetails, isFalse);
    await second;
    await site.getExtensions(query: 'cache-test');
    expect(transport.urls.length, 1);
    await site.getExtensions(query: 'cache-test', forceRefresh: true);
    expect(transport.urls.length, 2);
  });

  test('failed lists are not cached', () async {
    final transport = _Transport()..onProxy = (_) => Future.error(StateError('offline'));
    api.setTransportForTesting(transport);
    final site = GopeedSiteApi.instance;
    await expectLater(site.getExtensions(query: 'error-test'), throwsStateError);
    transport.onProxy = (_) async => _response({
      'data': [
        {'id': 'recovered'},
      ],
    });
    expect((await site.getExtensions(query: 'error-test')).data.single.id, 'recovered');
    expect(transport.urls.length, 2);
  });

  test('detail IDs are encoded once and a null README is still loaded detail', () async {
    final transport = _Transport()..onProxy = (_) async => _response({'id': '开发者@name', 'readme': null});
    api.setTransportForTesting(transport);
    final result = await GopeedSiteApi.instance.getExtension('开发者@name', version: '1.2.3');
    final uri = Uri.parse(transport.urls.single);
    expect(uri.pathSegments, ['api', 'extensions', '开发者@name']);
    expect(uri.queryParameters['version'], '1.2.3');
    expect(result.hasDetails, isTrue);
  });
}

Response<String> _response(Map<String, dynamic> json) => Response(
  requestOptions: RequestOptions(path: '/test'),
  data: jsonEncode(json),
  statusCode: 200,
);

class _Transport implements GopeedTransport {
  final urls = <String>[];
  Future<Response<String>> Function(String) onProxy = (_) async => _response({'data': []});

  @override
  Future<Response<String>> proxyRequest(String uri, {dynamic data, Options? options}) {
    urls.add(uri);
    return onProxy(uri);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
