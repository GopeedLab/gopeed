import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/create_task_batch.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/features/tasks/application/pending_download.dart';

PendingDownload source(String id, {String name = 'bundle', String rid = '', int count = 2, String? url}) =>
    PendingDownload(
        id: id,
        request: Request(url: url ?? 'https://example.com/$id'),
        options: Options(path: '/downloads'),
      )
      ..result = ResolveResult(
        id: rid,
        res: Resource(
          name: name,
          files: List.generate(
            count,
            (i) => FileInfo(
              name: '$i.txt',
              size: 100 + i,
              req: Request(url: 'https://example.com/$id/$i'),
            ),
          ),
        ),
      )
      ..selected.addAll(List.generate(count, (i) => i));

GopeedService service(CapabilityRegistry registry) => GopeedService(LocalCapabilityInvoker(registry));

void main() {
  test('one named extension group and all independent sources produce exactly two batches', () {
    final group = source('a')..selected.remove(0);
    final unnamed = source('b', name: '');
    final torrent = source('c', rid: 'bt-rid')..selected.remove(0);
    final http = source('d', rid: 'http-rid', count: 1, name: '');
    final batches = planDownloads([group, unnamed, torrent, http]);
    expect(batches, hasLength(2));
    final grouped = batches.first.request;
    expect(grouped.group, isTrue);
    expect(grouped.groupName, 'bundle');
    expect(grouped.opts!.path, '/downloads');
    expect(grouped.reqs!.single.req!.url, 'https://example.com/a/1');
    expect(grouped.toJson(), isNot(contains('groupId')));
    final normal = batches.last.request;
    expect(normal.group, isFalse);
    expect(normal.reqs, hasLength(4));
    expect(normal.reqs!.take(2).every((item) => item.req != null && item.rid == null), isTrue);
    expect(normal.reqs![2].rid, 'bt-rid');
    expect(normal.reqs![2].opts!.selectFiles, [1]);
    expect(normal.reqs![3].rid, 'http-rid');
    expect(normal.reqs![3].req, isNull);
  });

  test('successful batches are complete even when only some task IDs are returned', () async {
    var calls = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTaskBatch, (_) {
        calls++;
        return CreateTaskBatchResult(taskIds: ['only-one']);
      });
    final sources = [source('a', name: ''), source('b', rid: 'rid')];
    await planDownloads(sources).single.submit(service(registry));
    expect(sources.every((s) => s.created && s.createError == null), isTrue);
    expect(planDownloads(sources), isEmpty);
    expect(calls, 1);
  });

  test('retry only submits failed batches and never appends to the successful group', () async {
    final submitted = <CreateTaskBatch>[];
    var failNormal = true;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        submitted.add(batch);
        if (!batch.group && failNormal) throw StateError('all failed');
        return CreateTaskBatchResult(taskIds: ['created'], parentId: batch.group ? 'parent' : null);
      });
    final sources = [source('group'), source('normal', name: '')];
    for (final batch in planDownloads(sources)) {
      await batch.submit(service(registry));
    }
    expect(sources.first.created, isTrue);
    expect(sources.last.created, isFalse);
    expect(sources.last.createError, contains('all failed'));
    failNormal = false;
    final retry = planDownloads(sources);
    expect(retry, hasLength(1));
    await retry.single.submit(service(registry));
    expect(submitted.where((batch) => batch.group), hasLength(1));
    expect(planDownloads(sources), isEmpty);
  });

  test('single extension keeps nested resource directory and empty base remains empty', () {
    final single = source('single', count: 1, name: 'album');
    single.result!.res.files.single.path = 'disc1';
    expect(single.isGroup, isFalse);
    expect(planDownloads([single]).single.request.reqs!.single.opts!.path, '/downloads/album/disc1');
    final plain = source('plain', count: 1, name: '')..options.path = '';
    expect(plain.creationItems().single.opts!.path, '');
  });

  for (final parent in ['', 'downloads', './downloads', '/downloads']) {
    test('group members use paths relative to their group for parent "$parent"', () {
      final pending = source('group')..options.path = parent;
      pending.files[1].path = 'nested';
      final batch = planDownloads([pending]).single.request;
      expect(batch.group, isTrue);
      expect(batch.opts!.path, parent);
      expect(batch.reqs!.map((item) => item.opts!.path), ['', 'nested']);
      expect(pending.options.path, parent);
    });
  }

  test('unnamed extension members retain their ordinary batch destination', () {
    final pending = source('independent', name: '')..options.path = 'downloads';
    pending.files[1].path = 'nested';
    final batch = planDownloads([pending]).single.request;
    expect(batch.group, isFalse);
    expect(batch.reqs!.map((item) => item.opts!.path), ['downloads', 'downloads/nested']);
  });

  test('members preserve directory intent, HTTP sizes and per-file checksum rules', () {
    final pending = source('group');
    pending.options.extra = OptsExtraHttp(
      checksum: ChecksumOption(algorithm: 'md5', expected: 'abc'),
    ).toJson();
    final members = pending.creationItems();
    expect(members.first.opts!.size, 100);
    expect(members.first.opts!.asDefaultPath, isFalse);
    expect(members.first.opts!.skipAutoCategorize, isTrue);
    expect(members.first.opts!.extra as Map, isNot(contains('checksum')));
    pending.selected.remove(0);
    expect(pending.creationItems().single.opts!.extra as Map, contains('checksum'));
    expect(pending.options.extra as Map, contains('checksum'));
    pending.result!.res.files[1].req = Request(url: 'magnet:?xt=urn:btih:test');
    expect(pending.creationItems().single.opts!.size, 0);
  });

  test('FTP extension members use FTP options and keep their request TLS mode', () {
    final pending = source('group');
    pending.options.extra = OptsExtraHttp(
      connections: 64,
      autoExtract: true,
      archivePassword: 'secret',
      checksum: ChecksumOption(algorithm: 'md5', expected: 'abc'),
    ).toJson();
    pending.files[0].req = Request(
      url: 'ftp://files.example/archive.zip',
      extra: ReqExtraFtp(tls: 'explicit').toJson(),
    );
    pending.selected.remove(1);
    final item = pending.creationItems().single;
    expect(item.req!.extra, {'tls': 'explicit'});
    expect(item.opts!.size, 0);
    final extra = item.opts!.extra as Map;
    expect(extra['connections'], 32);
    expect(extra['autoExtract'], isTrue);
    expect(extra['archivePassword'], 'secret');
    expect(extra.containsKey('checksum'), isFalse);
    expect((pending.options.extra as Map)['connections'], 64);
  });

  test('member option changes do not affect other members or source options', () {
    final pending = source('group');
    pending.options
      ..name = 'custom'
      ..size = 500
      ..selectFiles = [0, 1]
      ..asDefaultPath = true;
    final original = pending.options.toJson();
    final members = pending.creationItems();
    members.first.opts!
      ..name = 'changed.txt'
      ..path = '/elsewhere'
      ..size = 1
      ..selectFiles.add(99);
    expect(members.last.opts!.name, '1.txt');
    expect(members.last.opts!.path, '');
    expect(members.last.opts!.size, 101);
    expect(members.last.opts!.selectFiles, [0, 1]);
    expect(pending.options.toJson(), original);
  });

  for (final rid in ['', 'resolved-rid']) {
    test('empty resource preserves options and selects the creation method for rid="$rid"', () {
      final pending = source('empty', rid: rid, count: 0)..selected.add(0);
      pending.options
        ..name = 'custom'
        ..selectFiles = [2];
      final item = pending.creationItems().single;
      expect(item.rid, rid.isEmpty ? null : rid);
      expect(item.req, rid.isEmpty ? pending.request : null);
      expect(identical(item.opts, pending.options), isFalse);
      expect(item.opts!.toJson(), pending.options.toJson());
    });
  }

  test('a magnet resolved by an extension keeps its HTTP members and group identity', () {
    final pending = source('extension', url: 'magnet:?xt=urn:btih:infohash');
    expect(pending.isBt, isFalse);
    expect(pending.isGroup, isTrue);
    final batch = planDownloads([pending]).single.request;
    expect(batch.group, isTrue);
    expect(batch.reqs, hasLength(2));
    expect(batch.reqs!.every((item) => item.rid == null && item.req!.url.startsWith('https:')), isTrue);
  });

  test('root names use HTTP file names, BT resource names and URL fallback', () {
    final http = source('http', rid: 'http-rid', count: 1, name: 'unused-resource-name');
    expect(http.name, '0.txt');
    final bt = source('bt', rid: 'bt-rid', count: 1, name: 'torrent', url: 'magnet:?xt=urn:btih:infohash');
    bt.result!.res.hash = 'infohash';
    expect(bt.isBt, isTrue);
    expect(bt.name, 'torrent');
    final unnamed = source('empty', rid: 'rid', count: 0, name: '');
    expect(unnamed.name, unnamed.request.url);
  });

  for (final entry in <String, bool>{
    'ed2k://|file|payload.bin|4|0123456789ABCDEF0123456789ABCDEF|/': false,
    'https://example.com/file': false,
    'https://example.com/file.torrent': false,
    'ftp://files.example/file.torrent': false,
    'ftps://files.example/file.torrent': false,
    'FTPES://files.example/file.torrent': false,
    'magnet:?xt=urn:btih:infohash': true,
    'MAGNET:?xt=urn:btih:infohash': true,
    'file:///downloads/file.torrent': true,
    '/downloads/FILE.TORRENT': true,
    'file:///downloads/file.torrent?query=1': true,
    'data:application/x-bittorrent;base64,AAAA': true,
    'data:application/octet-stream;base64,AAAA': false,
  }.entries) {
    test('source protocol determines BT identity for ${entry.key}', () {
      final pending = source('protocol', rid: 'rid', count: 1, name: 'resource', url: entry.key);
      if (RegExp(r'^ftp', caseSensitive: false).hasMatch(entry.key)) pending.result!.res.name = '';
      pending.result!.res.hash = '0123456789ABCDEF0123456789ABCDEF';
      expect(pending.isBt, entry.value);
      expect(pending.name, entry.value ? 'resource' : '0.txt');
    });
  }

  test('failed resolutions are excluded and each newly opened source resolves fresh', () async {
    var calls = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (_) {
        calls++;
        return ResolveResult(
          id: 'rid-$calls',
          res: Resource(files: [FileInfo(name: 'file')]),
        );
      });
    final first = PendingDownload(
      id: 'a',
      request: Request(url: 'https://example.com'),
      options: Options(),
    );
    await first.resolve(service(registry));
    await first.resolve(service(registry));
    expect(calls, 1);
    final reopened = PendingDownload(id: 'a', request: first.request, options: Options());
    await reopened.resolve(service(registry));
    expect(calls, 2);
    final failed = PendingDownload(id: 'b', request: first.request, options: Options())..resolveError = 'failed';
    expect(planDownloads([failed, reopened]).single.sources, [reopened]);
  });
}
