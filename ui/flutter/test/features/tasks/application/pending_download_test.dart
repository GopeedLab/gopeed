import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/create_task_batch.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/features/tasks/application/pending_download.dart';

PendingDownload source(String id, {String name = 'bundle', String rid = '', int count = 2}) =>
    PendingDownload(
        id: id,
        request: Request(url: 'https://example.com/$id'),
        options: Options(path: '/downloads'),
      )
      ..result = ResolveResult(
        id: rid,
        res: Resource(
          name: name,
          files: List.generate(
            count,
            (index) => FileInfo(
              name: '$index.txt',
              req: Request(url: 'https://example.com/$id/$index'),
            ),
          ),
        ),
      )
      ..selected.addAll(List.generate(count, (index) => index));

GopeedService service(CapabilityRegistry registry) => GopeedService(LocalCapabilityInvoker(registry));

void main() {
  test('grouping uses original extension resource, including one selected file', () async {
    final submitted = <CreateTaskBatch>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        submitted.add(batch);
        return CreateTaskBatchResult(
          taskIds: ['created'],
          groupId: 'server-id',
          items: [CreateTaskBatchItemResult(index: 0, taskId: 'created')],
        );
      });
    final pending = source('a')..selected.remove(0);
    await pending.submit(service(registry));
    expect(pending.isGroup, isTrue);
    expect(submitted.single.group, isTrue);
    expect(submitted.single.groupName, 'bundle');
    expect(submitted.single.toJson().containsKey('groupId'), isFalse);
    expect(submitted.single.opts!.path, '/downloads');
    expect(pending.createdFiles.keys, [1]);
    expect(source('unnamed', name: '').isGroup, isFalse);
    expect(source('single', count: 1).isGroup, isFalse);
    expect(source('bt', rid: 'rid').isGroup, isFalse);
  });

  test('named single-file extension keeps the resource directory', () async {
    CreateTask? submitted;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTask, (task) {
        submitted = task;
        return 'created';
      });
    final pending =
        PendingDownload(
            id: 'single',
            request: Request(url: 'https://example.com/folder'),
            options: Options(path: '/downloads'),
          )
          ..result = ResolveResult(
            res: Resource(
              name: 'album',
              files: [
                FileInfo(
                  name: 'song.mp3',
                  path: 'disc1',
                  req: Request(url: 'https://example.com/song'),
                ),
              ],
            ),
          )
          ..selected.add(0);
    await pending.submit(service(registry));
    expect(pending.isGroup, isFalse);
    expect(submitted!.opts!.path, '/downloads/album/disc1');
  });

  test('partial group retry submits only failed indexes without a caller group ID', () async {
    final batches = <CreateTaskBatch>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        batches.add(batch);
        final response = batches.length == 1
            ? CreateTaskBatchResult(
                taskIds: ['first'],
                groupId: 'one',
                items: [
                  CreateTaskBatchItemResult(index: 0, taskId: 'first'),
                  CreateTaskBatchItemResult(index: 1, error: 'failure'),
                ],
              )
            : CreateTaskBatchResult(
                taskIds: ['second'],
                groupId: 'two',
                items: [CreateTaskBatchItemResult(index: 0, taskId: 'second')],
              );
        return response;
      });
    final pending = source('a');
    pending.options.extra = OptsExtraHttp(
      checksum: ChecksumOption(algorithm: 'md5', expected: 'test'),
    ).toJson();
    await pending.submit(service(registry));
    expect(pending.createError, contains('failure'));
    pending.selected
      ..clear()
      ..add(1);
    await pending.submit(service(registry));
    await pending.submit(service(registry));
    expect(batches.length, 2);
    expect(batches.last.reqs!.single.req!.url, endsWith('/1'));
    expect(batches.last.toJson().containsKey('groupId'), isFalse);
    expect((batches.last.reqs!.single.opts!.extra as Map)['checksum'], isNull);
    expect(pending.createdFiles, {0: 'first', 1: 'second'});
    expect(pending.createError, isNull);
  });

  test('persisted member with start error is not recreated', () async {
    var calls = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTaskBatch, (_) {
        calls++;
        return CreateTaskBatchResult(
          taskIds: ['created'],
          groupId: 'group',
          items: [CreateTaskBatchItemResult(index: 0, taskId: 'created', error: 'start failed')],
        );
      });
    final pending = source('a')..selected.remove(1);
    await pending.submit(service(registry));
    await pending.submit(service(registry));
    expect(calls, 1);
    expect(pending.createError, isNull);
  });

  test('ordinary multi-file extensions preserve successful items without becoming groups', () async {
    final urls = <String>[];
    var fail = true;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTask, (request) {
        urls.add(request.req!.url);
        if (request.req!.url.endsWith('/1') && fail) throw StateError('failure');
        return request.req!.url;
      });
    final pending = source('a', name: '');
    await pending.submit(service(registry));
    expect(pending.createdFiles.keys, [0]);
    fail = false;
    await pending.submit(service(registry));
    expect(urls.map((url) => url.split('/').last), ['0', '1', '1']);
  });

  test('split members do not carry the parent default-directory flag', () async {
    CreateTask? submitted;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTask, (task) {
        submitted = task;
        return 'created';
      });
    final pending = source('ordinary', name: '')..options.asDefaultPath = true;
    await pending.submit(service(registry));
    expect(submitted!.opts!.path, '/downloads');
    expect(submitted!.opts!.asDefaultPath, isFalse);
  });

  test('direct BT keeps all-file semantics while resolved BT submits original selection once', () async {
    final tasks = <CreateTask>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTask, (request) {
        tasks.add(request);
        return 'created';
      });
    final direct = PendingDownload(
      id: 'direct',
      request: Request(url: 'magnet:?xt=urn:btih:hash'),
      options: Options(),
    )..selected.add(0);
    await direct.submit(service(registry), direct: true);
    expect(tasks.single.opts!.selectFiles, isEmpty);
    final bt = source('bt', rid: 'bt-rid', count: 3)..selected.remove(1);
    await bt.submit(service(registry));
    await bt.submit(service(registry));
    expect(tasks.length, 2);
    expect(tasks.last.rid, 'bt-rid');
    expect(tasks.last.opts!.selectFiles, [0, 2]);
  });

  test('multiple selected files clear checksum and unselected failures can be skipped', () async {
    CreateTaskBatch? submitted;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        submitted = batch;
        return CreateTaskBatchResult(
          taskIds: [],
          groupId: 'g',
          items: [
            CreateTaskBatchItemResult(index: 0, error: 'failure'),
            CreateTaskBatchItemResult(index: 1, error: 'failure'),
          ],
        );
      });
    final pending = source('a');
    pending.options.extra = OptsExtraHttp(
      checksum: ChecksumOption(algorithm: 'md5', expected: 'test'),
    ).toJson();
    await pending.submit(service(registry));
    for (final item in submitted!.reqs!) {
      expect((item.opts!.extra as Map)['checksum'], isNull);
    }
    pending.selected.clear();
    await pending.submit(service(registry));
    expect(pending.createError, isNull);
  });

  test('resolve failure can retry independently without repeating successful resolutions', () async {
    var calls = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (_) {
        if (++calls == 1) throw StateError('resolve failed');
        return ResolveResult(
          id: 'resolved',
          res: Resource(files: [FileInfo(name: 'file')]),
        );
      });
    final pending = PendingDownload(
      id: 'source',
      request: Request(url: 'https://example.com'),
      options: Options(),
    );
    await pending.resolve(service(registry));
    expect(pending.resolveError, isNotNull);
    await pending.resolve(service(registry));
    await pending.resolve(service(registry));
    expect(calls, 2);
    expect(pending.resolveError, isNull);
    expect(pending.selected, {0});
  });
}
