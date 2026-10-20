import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/network/mirror_client.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  late _CandidatesTransport transport;
  late Dio dio;
  late _Adapter adapter;
  late MirrorClient client;
  const resourceUrl = 'https://raw.githubusercontent.com/owner/repo/main/manifest.json';
  const mirror = 'https://mirror.example/https://raw.githubusercontent.com/owner/repo/main/manifest.json';
  setUp(() {
    transport = _CandidatesTransport([resourceUrl, mirror]);
    api.setTransportForTesting(transport);
    dio = Dio();
    adapter = _Adapter();
    dio.httpClientAdapter = adapter;
    client = MirrorClient(service: LocalAppCapabilities.instance.capabilities.gopeed, provider: 'github', client: dio);
  });
  tearDown(() {
    dio.close(force: true);
    for (final request in adapter.requests) {
      expect(request.headers.keys.map((key) => key.toLowerCase()), isNot(contains('x-api-token')));
    }
  });

  test('an invalid fast mirror cannot hide a slower valid release list', () async {
    final official = Completer<ResponseBody>();
    final mirrorRead = Completer<void>();
    adapter.reply = (request, _) {
      if (request.uri.host == 'raw.githubusercontent.com') return official.future;
      mirrorRead.complete();
      return Future.value(_json('<html>mirror error</html>'));
    };
    final result = client.getJson(resourceUrl, decode: _decodeList);
    await mirrorRead.future;
    official.complete(
      _json([
        {'tag_name': 'v2.0.0', 'body': 'notes'},
      ]),
    );
    final info = await result;
    expect(info.single['tag_name'], 'v2.0.0');
    expect(info.single['body'], 'notes');
    expect(adapter.requests.any((r) => r.uri.host == 'gopeed.com'), isFalse);
    expect(transport.requested, [resourceUrl]);
  });

  test('a complete validated winner cancels a stalled candidate', () async {
    final officialStarted = Completer<void>();
    final canceled = Completer<void>();
    adapter.reply = (request, cancellation) async {
      if (request.uri.host == 'raw.githubusercontent.com') {
        officialStarted.complete();
        await cancellation;
        canceled.complete();
        throw StateError('canceled');
      }
      await officialStarted.future;
      return _json([
        {'tag_name': 'v2.0.0'},
      ]);
    };
    expect((await client.getJson(resourceUrl, decode: _decodeList)).single['tag_name'], 'v2.0.0');
    await canceled.future;
    expect(adapter.requests.length, 2);
  });

  test('the mirror client supports other providers and non-GitHub resources', () async {
    const resource = 'https://resources.example/manifest.json';
    transport.provider = 'resources';
    transport.urls = [resource, 'https://cdn.example/manifest.json'];
    adapter.reply = (request, _) async =>
        request.uri.host == 'resources.example' ? _json({}, status: 503) : _json({'revision': 42});
    final client = MirrorClient(
      service: LocalAppCapabilities.instance.capabilities.gopeed,
      provider: 'resources',
      client: dio,
    );
    final revision = await client.getJson<int>(resource, decode: (data) => (data as Map)['revision'] as int);
    expect(revision, 42);
    expect(transport.requested, [resource]);
    expect(adapter.requests.map((request) => request.uri.host).toSet(), {'resources.example', 'cdn.example'});
  });

  test('metadata attempts have a total timeout even if the transport stalls', () async {
    final canceled = Completer<void>();
    adapter.reply = (_, cancellation) async {
      await cancellation;
      canceled.complete();
      throw StateError('canceled');
    };
    transport.urls = [resourceUrl];
    final github = MirrorClient(
      service: LocalAppCapabilities.instance.capabilities.gopeed,
      provider: 'github',
      client: dio,
      timeout: const Duration(milliseconds: 30),
    );
    await expectLater(github.getJson(resourceUrl, decode: (data) => data), throwsA(isA<TimeoutException>()));
    await canceled.future;
  });
}

List<dynamic> _decodeList(dynamic data) {
  if (data is! List) throw const FormatException('Invalid manifest');
  return data;
}

ResponseBody _json(dynamic body, {int status = 200}) => ResponseBody.fromString(
  jsonEncode(body),
  status,
  headers: {
    Headers.contentTypeHeader: [Headers.jsonContentType],
  },
);

class _Adapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  late Future<ResponseBody> Function(RequestOptions, Future<void>?) reply;
  @override
  Future<ResponseBody> fetch(RequestOptions options, Stream<List<int>>? requestStream, Future<void>? cancelFuture) {
    requests.add(options);
    return reply(options, cancelFuture);
  }

  @override
  void close({bool force = false}) {}
}

class _CandidatesTransport implements GopeedTransport {
  _CandidatesTransport(this.urls);
  List<String> urls;
  String provider = 'github';
  bool fail = false;
  final requested = <String>[];
  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    expect(path, 'api/v1/mirrors/$provider/candidates');
    expect(method, 'POST');
    requested.add((data as Map)['url'] as String);
    if (fail) throw StateError('Downloader unavailable');
    return {
      'code': 0,
      'data': {'urls': urls},
    };
  }

  @override
  String join(String path) => path;
}
