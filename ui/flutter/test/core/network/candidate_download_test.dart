import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:crypto/crypto.dart';
import 'package:gopeed/core/network/candidate_download.dart';

void main() {
  final expectedSha256 = sha256.convert([for (var i = 1; i <= 3; i++) ...List.filled(32768, i)]).toString();
  late HttpServer server;
  var requests = 0;
  late Directory directory;
  late String savePath;
  late List<String> downloads;
  late Future<void> Function(HttpRequest request) handle;

  setUp(() async {
    directory = await Directory.systemTemp.createTemp('gopeed-update-test-');
    savePath = '${directory.path}/package.zip';
    downloads = [];
    requests = 0;
    server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    server.listen((request) async {
      requests++;
      if (request.method == 'GET') downloads.add(request.uri.path);
      await handle(request);
      await request.response.close();
    });
  });

  tearDown(() async {
    await server.close(force: true);
    await directory.delete(recursive: true);
  });

  String url(String path) => 'http://127.0.0.1:${server.port}$path';

  Future<void> package(HttpRequest request) async {
    const chunkSize = 32768;
    request.response.headers.contentType = ContentType.binary;
    request.response.contentLength = chunkSize * 3;
    if (request.method == 'HEAD') return;
    for (var i = 0; i < 3; i++) {
      request.response.add(List.filled(chunkSize, i + 1));
      await request.response.flush();
      await Future<void>.delayed(const Duration(milliseconds: 20));
    }
  }

  test('downloads the fastest successful probe and reports intermediate byte progress', () async {
    handle = (request) async {
      if (request.uri.path == '/slow' && request.method == 'HEAD') {
        await Future<void>.delayed(const Duration(milliseconds: 120));
      }
      await package(request);
    };
    final progress = <(int, int)>[];
    await downloadFromCandidates(
      [url('/slow'), url('/fast'), url('/fast')],
      savePath,
      expectedSha256: expectedSha256,
      onReceiveProgress: (received, total) => progress.add((received, total)),
    );
    expect(downloads, ['/fast']);
    expect(await File(savePath).length(), 98304);
    expect(progress.any((p) => p.$1 > 0 && p.$1 < p.$2), isTrue);
    expect(progress.last, (98304, 98304));
  });

  test('falls back when a successful HEAD is followed by a failed GET', () async {
    handle = (request) async {
      if (request.uri.path == '/broken') {
        request.response.headers.contentType = ContentType.binary;
        if (request.method == 'GET') {
          request.response.statusCode = HttpStatus.serviceUnavailable;
          request.response.write('unavailable');
        }
        return;
      }
      if (request.method == 'HEAD') {
        await Future<void>.delayed(const Duration(milliseconds: 60));
      }
      await package(request);
    };
    await downloadFromCandidates(
      [url('/broken'), url('/working')],
      savePath,
      expectedSha256: expectedSha256,
      onReceiveProgress: (_, _) {},
    );
    expect(downloads, ['/broken', '/working']);
    final bytes = await File(savePath).readAsBytes();
    expect(bytes.length, 98304);
    expect(bytes.take(32768), everyElement(1));
  });

  test('tries GET even when HEAD is unsupported', () async {
    handle = (request) async {
      if (request.method == 'HEAD') {
        request.response.statusCode = HttpStatus.methodNotAllowed;
        return;
      }
      await package(request);
    };
    await downloadFromCandidates(
      [url('/no-head')],
      savePath,
      expectedSha256: expectedSha256,
      onReceiveProgress: (_, _) {},
    );
    expect(downloads, ['/no-head']);
    expect(await File(savePath).length(), 98304);
  });

  test('bounds probe time and still allows a normal download', () async {
    handle = (request) async {
      if (request.method == 'HEAD') {
        await Future<void>.delayed(const Duration(milliseconds: 100));
        return;
      }
      await package(request);
    };
    await downloadFromCandidates(
      [url('/head-timeout')],
      savePath,
      expectedSha256: expectedSha256,
      probeTimeout: const Duration(milliseconds: 20),
      onReceiveProgress: (_, _) {},
    );
    expect(downloads, ['/head-timeout']);
    expect(await File(savePath).length(), 98304);
  });

  test('rejects mirror landing pages and removes an unsuccessful download', () async {
    handle = (request) async {
      request.response.headers.contentType = ContentType.html;
      if (request.method == 'GET') request.response.write('<html>mirror error</html>');
    };
    await expectLater(
      downloadFromCandidates([url('/landing')], savePath, expectedSha256: expectedSha256, onReceiveProgress: (_, _) {}),
      throwsStateError,
    );
    expect(await File(savePath).exists(), isFalse);
  });
  test('same-sized modified mirror bytes fail the official hash and fall back', () async {
    handle = (request) async {
      if (request.uri.path == '/tampered') {
        request.response.headers.contentType = ContentType.binary;
        // A mirror-provided hash must never replace the expected official hash.
        request.response.headers.set('x-checksum-sha256', sha256.convert(List.filled(98304, 9)).toString());
        request.response.contentLength = 98304;
        if (request.method == 'GET') request.response.add(List.filled(98304, 9));
        return;
      }
      if (request.method == 'HEAD') await Future<void>.delayed(const Duration(milliseconds: 60));
      await package(request);
    };
    await downloadFromCandidates(
      [url('/tampered'), url('/official')],
      savePath,
      expectedSha256: expectedSha256,
      onReceiveProgress: (_, _) {},
    );
    expect(downloads, ['/tampered', '/official']);
    expect((await sha256.bind(File(savePath).openRead()).first).toString(), expectedSha256);
  });

  test('all checksum failures abort and delete the untrusted package', () async {
    handle = (request) async {
      request.response.headers.contentType = ContentType.binary;
      if (request.method == 'GET') request.response.add(List.filled(98304, 9));
    };
    await expectLater(
      downloadFromCandidates(
        [url('/bad-one'), url('/bad-two')],
        savePath,
        expectedSha256: expectedSha256,
        onReceiveProgress: (_, _) {},
      ),
      throwsA(isA<DownloadChecksumException>()),
    );
    expect(downloads.toSet(), {'/bad-one', '/bad-two'});
    expect(await File(savePath).exists(), isFalse);
  });

  test('missing or malformed official checksums cannot start a probe or download', () async {
    handle = package;
    for (final checksum in ['', 'sha256:$expectedSha256', '0' * 63, 'g' * 64, '$expectedSha256\n']) {
      await expectLater(
        downloadFromCandidates([url('/official')], savePath, expectedSha256: checksum, onReceiveProgress: (_, _) {}),
        throwsA(isA<DownloadChecksumException>()),
      );
    }
    expect(requests, 0);
    expect(await File(savePath).exists(), isFalse);
  });
}
