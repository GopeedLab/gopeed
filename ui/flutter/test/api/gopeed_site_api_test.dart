import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/gopeed_site_api.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  test('summary requests use the business API and remain shared and cached', () async {
    final pending = Completer<Map<String, dynamic>>();
    final transport = _Transport()..onRequest = (_) => pending.future;
    api.setTransportForTesting(transport);
    final site = GopeedSiteApi.instance;
    final first = site.getExtensions(query: 'cache-test', forceRefresh: true);
    final second = site.getExtensions(query: 'cache-test');
    expect(transport.urls.length, 1);
    expect(Uri.parse(transport.urls.single).path, 'api/v1/store/extensions');
    pending.complete({
      'data': [
        {'id': 'test'},
      ],
    });
    expect((await first).data.single.hasDetails, isFalse);
    await second;
    await site.getExtensions(query: 'cache-test');
    expect(transport.urls.length, 1);
    await site.getExtensions(query: 'cache-test', forceRefresh: true);
    expect(transport.urls.length, 2);
  });
  test('failed lists are not cached', () async {
    final transport = _Transport()..onRequest = (_) => Future.error(StateError('offline'));
    api.setTransportForTesting(transport);
    await expectLater(GopeedSiteApi.instance.getExtensions(query: 'error-test'), throwsStateError);
    transport.onRequest = (_) async => {
      'data': [
        {'id': 'recovered'},
      ],
    };
    expect((await GopeedSiteApi.instance.getExtensions(query: 'error-test')).data.single.id, 'recovered');
    expect(transport.urls.length, 2);
  });
  test('detail IDs are encoded once and versions use business parameters', () async {
    final transport = _Transport()..onRequest = (_) async => {'id': '开发者@name', 'readme': null};
    api.setTransportForTesting(transport);
    final result = await GopeedSiteApi.instance.getExtension('开发者@name', version: '1.2.3');
    final uri = Uri.parse(transport.urls.single);
    expect(uri.pathSegments, ['api', 'v1', 'store', 'extensions', '开发者@name']);
    expect(uri.queryParameters['version'], '1.2.3');
    expect(result.hasDetails, isTrue);
  });
}

class _Transport implements GopeedTransport {
  final urls = <String>[];
  Future<Map<String, dynamic>> Function(String) onRequest = (_) async => {'data': []};
  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    final uri = Uri.parse(
      path,
    ).replace(queryParameters: queryParameters?.map((key, value) => MapEntry(key, value.toString())));
    urls.add(uri.toString());
    return {'code': 0, 'data': await onRequest(uri.toString())};
  }

  @override
  String join(String path) => path;
}
