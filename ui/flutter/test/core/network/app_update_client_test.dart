import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/gopeed_site_api.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/network/app_update_client.dart';
import 'package:gopeed/core/network/mirror_client.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  late _CandidatesTransport transport;
  late Dio dio;
  late _Adapter adapter;
  late AppUpdateClient updates;
  const mirror = 'https://mirror.example/https://api.github.com/repos/GopeedLab/gopeed/releases?per_page=10';

  setUp(() {
    transport = _CandidatesTransport([AppUpdateClient.releasesUrl, mirror]);
    api.setTransportForTesting(transport);
    dio = Dio();
    adapter = _Adapter();
    dio.httpClientAdapter = adapter;
    updates = AppUpdateClient(
      github: MirrorClient(service: LocalAppCapabilities.instance.capabilities.gopeed, provider: 'github', client: dio),
      site: GopeedSiteApi(client: dio),
    );
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
      if (request.uri.host == 'api.github.com') return official.future;
      mirrorRead.complete();
      return Future.value(_json('<html>mirror error</html>'));
    };
    final result = updates.check('1.0.0');
    await mirrorRead.future;
    official.complete(
      _json([
        {'tag_name': 'v2.0.0', 'body': 'notes'},
      ]),
    );
    final info = await result;
    expect(info!.version, '2.0.0');
    expect(info.changeLog, 'notes');
    expect(info.releaseUrl, 'https://github.com/GopeedLab/gopeed/releases/tag/v2.0.0');
    expect(adapter.requests.any((r) => r.uri.host == 'gopeed.com'), isFalse);
    expect(transport.requested, [AppUpdateClient.releasesUrl]);
  });

  test('a complete validated winner cancels a stalled candidate', () async {
    final officialStarted = Completer<void>();
    final canceled = Completer<void>();
    adapter.reply = (request, cancellation) async {
      if (request.uri.host == 'api.github.com') {
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
    expect((await updates.check('1.0.0'))!.version, '2.0.0');
    await canceled.future;
    expect(adapter.requests.length, 2);
  });

  test('website fallback runs once only after every GitHub candidate fails', () async {
    adapter.reply = (request, _) async => switch (request.uri.host) {
      'api.github.com' => _json({'message': 'rate limited'}, status: 403),
      'mirror.example' => _json([
        {'message': 'not releases'},
      ]),
      _ => _json([
        {'tag_name': 'v2.0.0'},
      ]),
    };
    expect((await updates.check('1.0.0'))!.version, '2.0.0');
    final website = adapter.requests.where((r) => r.uri.host == 'gopeed.com').single;
    expect(website.method, 'GET');
    expect(website.uri.toString(), 'https://gopeed.com/api/releases?per_page=10');
  });

  test('candidate API failure still allows the independent website provider', () async {
    transport.fail = true;
    adapter.reply = (_, _) async => _json([
      {'tag_name': 'v2.0.0'},
    ]);
    expect((await updates.check('1.0.0'))!.version, '2.0.0');
    expect(adapter.requests.single.uri.host, 'gopeed.com');
  });

  test('valid empty and no-update lists do not fall back', () async {
    transport.urls = [AppUpdateClient.releasesUrl];
    for (final releases in [
      [],
      [
        {'tag_name': 'v1.0.0'},
      ],
    ]) {
      adapter.requests.clear();
      adapter.reply = (_, _) async => _json(releases);
      expect(await updates.check('1.0.0'), isNull);
      expect(adapter.requests.single.uri.host, 'api.github.com');
    }
  });

  test('null, objects, invalid releases and failed website responses propagate', () async {
    transport.urls = [AppUpdateClient.releasesUrl];
    for (final invalid in [
      null,
      {'message': 'error'},
      [
        {'tag_name': 'not a version'},
      ],
    ]) {
      adapter.requests.clear();
      adapter.reply = (_, _) async => _json(invalid);
      await expectLater(updates.check('1.0.0'), throwsFormatException);
      expect(adapter.requests.map((r) => r.uri.host), ['api.github.com', 'gopeed.com']);
    }
    adapter.reply = (_, _) async => _json({}, status: 503);
    await expectLater(updates.check('1.0.0'), throwsA(isA<DioException>()));
  });

  test('SemVer selection preserves stable and same-series preview rules', () async {
    transport.urls = [AppUpdateClient.releasesUrl];
    var releases = [
      {'tag_name': 'v2.1.0-beta.1', 'prerelease': true},
      {'tag_name': 'v2.0.0-beta.2', 'prerelease': true},
      {'tag_name': 'v2.0.0-rc.1', 'prerelease': true},
      {'tag_name': 'v1.9.9'},
      {'tag_name': 'v2.0.0', 'draft': true},
    ];
    adapter.reply = (_, _) async => _json(releases);
    expect((await updates.check('2.0.0-beta.1'))!.version, '2.0.0-rc.1');
    expect((await updates.check('1.9.8'))!.version, '1.9.9');
    releases = [
      {'tag_name': 'v2.0.0-beta.10'},
      {'tag_name': 'v2.0.0-beta.2'},
    ];
    expect((await updates.check('2.0.0-beta.1'))!.version, '2.0.0-beta.10');
    releases = [
      {'tag_name': 'v2.0.0'},
      {'tag_name': 'v2.0.1'},
    ];
    expect((await updates.check('2.0.0-beta.1'))!.version, '2.0.1');
    releases = [
      {'tag_name': 'v2.0.0+build.2'},
    ];
    expect(await updates.check('2.0.0+build.1'), isNull);
    await expectLater(updates.check('invalid'), throwsFormatException);
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
    transport.urls = [AppUpdateClient.releasesUrl];
    final github = MirrorClient(
      service: LocalAppCapabilities.instance.capabilities.gopeed,
      provider: 'github',
      client: dio,
      timeout: const Duration(milliseconds: 30),
    );
    await expectLater(
      github.getJson(AppUpdateClient.releasesUrl, decode: (data) => data),
      throwsA(isA<TimeoutException>()),
    );
    await canceled.future;
  });
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
