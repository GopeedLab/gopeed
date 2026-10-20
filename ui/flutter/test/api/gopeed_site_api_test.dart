import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/gopeed_site_api.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  late Dio client;
  late _SiteAdapter adapter;
  late GopeedSiteApi site;
  late _Transport transport;

  setUp(() {
    client = Dio();
    adapter = _SiteAdapter();
    client.httpClientAdapter = adapter;
    site = GopeedSiteApi(client: client);
    transport = _Transport();
    api.setTransportForTesting(transport);
  });
  tearDown(() {
    client.close(force: true);
    expect(transport.calls, 0);
    for (final request in adapter.requests) {
      expect(request.uri.origin, 'https://gopeed.com');
      expect(request.headers.keys.map((key) => key.toLowerCase()), isNot(contains('x-api-token')));
    }
  });

  test('install counts are reported directly with JSON and failures remain observable', () async {
    await site.reportExtensionInstall('开发者@name');
    final request = adapter.requests.single;
    expect(request.uri.toString(), 'https://gopeed.com/api/extensions/install');
    expect(request.method, 'POST');
    expect(request.contentType, Headers.jsonContentType);
    expect(request.data, {'id': '开发者@name'});
    adapter.statusCode = 503;
    await expectLater(site.reportExtensionInstall('test'), throwsA(isA<DioException>()));
  });

  test('summary requests go directly to the website and remain shared and cached', () async {
    final pending = Completer<Map<String, dynamic>>();
    final started = Completer<void>();
    adapter.onRequest = (_) {
      if (!started.isCompleted) started.complete();
      return pending.future;
    };
    final first = site.getExtensions(query: 'cache-test', forceRefresh: true);
    final second = site.getExtensions(query: 'cache-test');
    await started.future;
    expect(adapter.requests.length, 1);
    final request = adapter.requests.single;
    expect(request.method, 'GET');
    expect(request.uri.path, '/api/extensions');
    expect(request.uri.queryParameters, {
      'view': 'summary',
      'page': '1',
      'limit': '20',
      'sort': 'stars',
      'order': 'desc',
      'q': 'cache-test',
    });
    pending.complete({
      'data': [
        {'id': 'test'},
      ],
    });
    expect((await first).data.single.hasDetails, isFalse);
    await second;
    await site.getExtensions(query: 'cache-test');
    expect(adapter.requests.length, 1);
    await site.getExtensions(query: 'cache-test', forceRefresh: true);
    expect(adapter.requests.length, 2);
  });

  test('failed lists are not cached', () async {
    adapter.statusCode = 503;
    await expectLater(site.getExtensions(query: 'error-test'), throwsA(isA<DioException>()));
    adapter.statusCode = 200;
    adapter.onRequest = (_) async => {
      'data': [
        {'id': 'recovered'},
      ],
    };
    expect((await site.getExtensions(query: 'error-test')).data.single.id, 'recovered');
    expect(adapter.requests.length, 2);
  });

  test('detail IDs are encoded once and version remains a query parameter', () async {
    adapter.onRequest = (_) async => {'id': '开发者@name', 'readme': null};
    final result = await site.getExtension('开发者@name', version: '1.2.3');
    final uri = adapter.requests.single.uri;
    expect(uri.pathSegments, ['api', 'extensions', '开发者@name']);
    expect(uri.queryParameters['version'], '1.2.3');
    expect(result.hasDetails, isTrue);
    await site.getExtension('literal%2Fname');
    expect(adapter.requests.last.uri.pathSegments.last, 'literal%2Fname');
  });
}

class _Transport implements GopeedTransport {
  int calls = 0;
  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    calls++;
    throw StateError('Store requests must not use the Downloader transport');
  }

  @override
  String join(String path) => path;
}

class _SiteAdapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  int statusCode = 200;
  Future<Map<String, dynamic>> Function(RequestOptions) onRequest = (_) async => {'data': []};
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<List<int>>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    return ResponseBody.fromString(
      jsonEncode(await onRequest(options)),
      statusCode,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}
