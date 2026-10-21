import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/gopeed_site_api.dart';
import 'package:gopeed/core/network/app_update_client.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  late Dio dio;
  late _Adapter adapter;
  late AppUpdateClient updates;
  late _RejectTransport transport;
  setUp(() {
    transport = _RejectTransport();
    api.setTransportForTesting(transport);
    dio = Dio();
    adapter = _Adapter();
    dio.httpClientAdapter = adapter;
    updates = AppUpdateClient(
      client: dio,
      site: GopeedSiteApi(client: dio),
    );
  });
  tearDown(() {
    dio.close(force: true);
    expect(transport.calls, 0);
    for (final request in adapter.requests) {
      expect(request.headers.keys.map((key) => key.toLowerCase()), isNot(contains('x-api-token')));
    }
  });

  test('update metadata comes directly from the official GitHub API without mirror lookup', () async {
    adapter.reply = (_, _) async => _json([
      {'tag_name': 'v2.0.0', 'body': 'notes'},
    ]);
    final info = await updates.check('1.0.0');
    expect(info!.version, '2.0.0');
    expect(info.changeLog, 'notes');
    expect(info.releaseUrl, 'https://github.com/GopeedLab/gopeed/releases/tag/v2.0.0');
    expect(adapter.requests.single.uri.toString(), AppUpdateClient.releasesUrl);
  });

  test('release asset hashes are bound to the selected release and exact file name', () async {
    final good = 'aB' * 32;
    adapter.reply = (_, _) async => _json([
      {
        'tag_name': 'v1.9.9',
        'assets': [
          {'name': 'old.zip', 'digest': 'sha256:${'c' * 64}'},
        ],
      },
      {
        'tag_name': 'v2.0.0',
        'assets': [
          {'name': 'package.zip', 'digest': 'sha256:$good'},
          {'name': 'wrong-algorithm.zip', 'digest': 'md5:${'a' * 32}'},
          {'name': 'missing.zip', 'digest': null},
          {'name': 'bad-hex.zip', 'digest': 'sha256:${'g' * 64}'},
          {'name': 'short.zip', 'digest': 'sha256:1234'},
        ],
      },
    ]);
    final info = (await updates.check('1.0.0'))!;
    expect(info.assetSha256, {'package.zip': good.toLowerCase()});
    expect(() => info.assetSha256['package.zip'] = 'd' * 64, throwsUnsupportedError);
    expect(adapter.requests.single.followRedirects, isFalse);
  });

  test('the official website fallback preserves asset digests', () async {
    final checksum = 'e' * 64;
    adapter.reply = (request, _) async => request.uri.host == 'api.github.com'
        ? _json({'message': 'rate limited'}, status: 403)
        : _json([
            {
              'tag_name': 'v2.0.0',
              'assets': [
                {'name': 'package.dmg', 'digest': 'sha256:$checksum'},
              ],
            },
          ]);
    final info = (await updates.check('1.0.0'))!;
    expect(info.assetSha256, {'package.dmg': checksum});
    expect(adapter.requests.map((request) => request.uri.host), ['api.github.com', 'gopeed.com']);
    expect(adapter.requests.every((request) => !request.followRedirects), isTrue);
  });

  test('API redirects fail instead of obtaining metadata from another host', () async {
    adapter.reply = (_, _) async => ResponseBody.fromString(
      '',
      302,
      headers: {
        'location': ['https://mirror.example/releases'],
      },
    );
    await expectLater(updates.check('1.0.0'), throwsA(isA<DioException>()));
    expect(adapter.requests.map((request) => request.uri.host), ['api.github.com', 'gopeed.com']);
    expect(adapter.requests.every((request) => !request.followRedirects), isTrue);
  });

  test('website fallback runs once after the official GitHub request fails', () async {
    adapter.reply = (request, _) async => switch (request.uri.host) {
      'api.github.com' => _json({'message': 'rate limited'}, status: 403),
      _ => _json([
        {'tag_name': 'v2.0.0'},
      ]),
    };
    expect((await updates.check('1.0.0'))!.version, '2.0.0');
    expect(adapter.requests.map((request) => request.uri.host), ['api.github.com', 'gopeed.com']);
    final website = adapter.requests.where((r) => r.uri.host == 'gopeed.com').single;
    expect(website.method, 'GET');
    expect(website.uri.toString(), 'https://gopeed.com/api/releases?per_page=10');
  });

  test('invalid GitHub metadata falls back to valid website metadata', () async {
    for (final invalid in [
      null,
      '<html>error</html>',
      {'message': 'error'},
      [
        {'tag_name': 'invalid'},
      ],
    ]) {
      adapter.requests.clear();
      adapter.reply = (request, _) async => request.uri.host == 'api.github.com'
          ? _json(invalid)
          : _json([
              {'tag_name': 'v2.0.0'},
            ]);
      expect((await updates.check('1.0.0'))!.version, '2.0.0');
      expect(adapter.requests.map((request) => request.uri.host), ['api.github.com', 'gopeed.com']);
    }
  });

  test('valid empty and no-update lists do not fall back', () async {
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

class _RejectTransport implements GopeedTransport {
  int calls = 0;
  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    calls++;
    throw StateError('Update checks must not use the Downloader or mirror API');
  }

  @override
  String join(String path) => path;
}
