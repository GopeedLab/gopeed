import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/task.dart' as api;
import 'package:gopeed/core/icons/gopeed_icons.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';

void main() {
  api.Task buildTask(List<int> selection, int size, {String? doneAt}) => api.Task.fromJson({
    'id': 'bt-selection',
    'name': 'bundle',
    'protocol': 'bt',
    'status': 'running',
    'uploading': false,
    'createdAt': '2026-01-01T00:00:00Z',
    'updatedAt': '2026-01-01T00:00:00Z',
    'doneAt': doneAt,
    'meta': {
      'req': {'url': 'magnet:?xt=urn:btih:test'},
      'opts': {'path': '/downloads', 'selectFiles': selection},
      'res': {
        'name': 'bundle',
        'size': size,
        'files': [
          {'name': 'a.txt', 'size': 10},
          {'name': 'b.txt', 'size': 20},
          {'name': 'c.txt', 'path': 'nested', 'size': 30},
        ],
      },
    },
    'progress': {'used': 0, 'speed': 0, 'downloaded': 10, 'uploadSpeed': 0, 'uploaded': 0},
  });

  test('groups decode fetcher-less records and hide only their own members', () {
    final group = groupTestTask('group', group: true);
    final records = taskRecordsFromApi([
      group,
      groupTestTask('first', parentId: 'group'),
      groupTestTask(
        'second',
        parentId: 'group',
        directory: '/downloads/folder/nested',
        name: 'movie.mp4',
        size: 300,
        downloaded: 75,
      ),
      groupTestTask('standalone'),
      groupTestTask('orphan', parentId: 'missing-group'),
    ]);
    expect(group.protocol, isNull);
    expect(group.meta.req.url, isEmpty);
    expect(records.map((record) => record.id), ['group', 'standalone', 'orphan']);
    final record = records.first;
    expect(record.isGroup, isTrue);
    expect(record.isFolder, isTrue);
    expect(record.members.length, 2);
    expect(record.totalBytes, 400);
    expect(record.downloadedBytes, 100);
    expect(record.speedBytes, 20);
    expect(record.progress, 0.25);
    expect(record.storagePath, '/downloads/folder');
    expect(record.files.map((file) => file.path), ['', 'nested']);
    expect(record.files.map((file) => file.name), ['file.txt', 'movie.mp4']);
    expect(record.files.last.downloadedBytes, 75);
    expect(record.matchesSearch('movie'), isTrue);
    expect(record.matchesSearch('second'), isTrue);
    expect(record.matchesSearch('absent'), isFalse);
    expect(record.canUpdateUrl, isFalse);
  });

  for (final scenario in <(List<String>, TaskStatus, bool)>[
    (['running', 'error', 'pause'], TaskStatus.downloading, false),
    (['error', 'pause', 'wait'], TaskStatus.failed, false),
    (['pause', 'wait', 'done'], TaskStatus.paused, false),
    (['done', 'done'], TaskStatus.completed, false),
    (['wait', 'done'], TaskStatus.downloading, true),
    ([], TaskStatus.paused, false),
  ]) {
    test('group status priority for ${scenario.$1}', () {
      final record = taskRecordsFromApi([
        groupTestTask('group', group: true),
        for (var i = 0; i < scenario.$1.length; i++)
          groupTestTask('child-$i', parentId: 'group', status: scenario.$1[i]),
      ]).single;
      expect(record.status, scenario.$2);
      expect(record.waiting, scenario.$3);
      if (scenario.$2 == TaskStatus.completed) {
        expect(record.progress, 1);
        expect(record.doneAt, DateTime.utc(2026, 1, 3));
      }
    });
  }

  test('unknown member sizes stay indeterminate and empty files can finish', () {
    final group = groupTestTask('group', group: true);
    final unknown = groupTestTask('unknown', parentId: 'group', size: 0);
    final record = taskRecordsFromApi([group, unknown, groupTestTask('known', parentId: 'group')]).single;
    expect(record.totalBytes, isNull);
    expect(record.progress, isNull);
    unknown.status = api.Status.done;
    final empty = taskRecordsFromApi([group, unknown]).single;
    expect(empty.totalBytes, 0);
    expect(empty.progress, 1);
  });

  test('Windows group directories retain their relative file tree', () {
    final records = taskRecordsFromApi([
      groupTestTask('group', group: true, directory: r'C:\Downloads\folder'),
      groupTestTask('child', parentId: 'group', directory: r'C:\Downloads\folder\nested'),
    ]);
    expect(records.single.files.single.path, 'nested');
    expect(records.single.files.single.name, 'file.txt');
  });

  test('selected files retain original indexes for runtime progress', () {
    final record = TaskRecord.fromApi(buildTask([2, 0], 40));
    expect(record.files.map((file) => file.name), ['a.txt', 'c.txt']);
    expect(record.files.map((file) => file.resourceIndex), [0, 2]);
    expect(record.files.last.path, 'nested');
    expect(record.totalBytes, 40);
    expect(record.progress, 0.25);
  });

  test('empty selection preserves default all files', () {
    final record = TaskRecord.fromApi(buildTask([], 60));
    expect(record.files.length, 3);
    expect(record.totalBytes, 60);
  });

  test('underestimated HTTP size caps progress without completing the task', () {
    final task = buildTask([], 3)..protocol = api.Protocol.http;
    task.progress.speed = 2;
    final record = TaskRecord.fromApi(task);
    expect(record.progress, 1);
    expect(record.status, TaskStatus.downloading);
    expect(record.downloadedBytes, 10);
    expect(record.totalBytes, 3);
    expect(record.remainingSeconds, isNull);
    expect(const TaskFileNode(path: '', name: 'video.mp4', sizeBytes: 3, downloadedBytes: 10).progress, 1);
  });

  test('queued task uses the creation estimate until a resolved size is available', () {
    final task = buildTask([], 0)..protocol = api.Protocol.http;
    task.meta.opts.size = 1024;
    task.meta.res = null;
    task.status = api.Status.wait;
    expect(TaskRecord.fromApi(task).totalBytes, 1024);
    task.meta.res = buildTask([], 0).meta.res;
    expect(TaskRecord.fromApi(task).totalBytes, isNull); // A confirmed zero must not fall back to 1024.
    task.meta.res!.size = 512;
    expect(TaskRecord.fromApi(task).totalBytes, 512);
  });

  test('completion time maps from api and stays null for legacy tasks', () {
    final completed = TaskRecord.fromApi(buildTask([], 60, doneAt: '2026-01-02T03:04:05Z'));
    expect(completed.doneAt, DateTime.parse('2026-01-02T03:04:05Z'));
    final legacy = TaskRecord.fromApi(buildTask([], 60));
    expect(legacy.doneAt, isNull);
  });

  test('file names preserve hash and question mark characters for type detection', () {
    expect(
      taskFileTypeIcon('Slow English Podcast #2 _ Eating, Talking, and Learning Natural English.mp4'),
      GopeedIcons.fileVideo,
    );
    expect(taskFileTypeIcon('What is this? #2.MP4'), GopeedIcons.fileVideo);
    expect(taskFileTypeIcon('Episode #2.mp3'), GopeedIcons.fileAudio);
  });

  test('URL fallback ignores query and fragment when detecting file type', () {
    expect(taskFileTypeIcon('https://example.com/video.mp4?token=abc#part'), GopeedIcons.fileVideo);
    expect(taskFileTypeIcon('https://example.com/download?name=video.mp4'), GopeedIcons.file);
    expect(taskFileTypeIcon('Playlist #2.mp4', isFolder: true), GopeedIcons.folder);
  });
}

api.Task groupTestTask(
  String id, {
  String parentId = '',
  bool group = false,
  String status = 'running',
  int size = 100,
  int downloaded = 25,
  String directory = '/downloads/folder',
  String name = 'file.txt',
}) => api.Task.fromJson({
  'id': id,
  'name': group ? 'folder' : name,
  'type': group ? 'group' : '',
  'parentId': parentId,
  'protocol': group ? '' : 'http',
  'status': status,
  'uploading': false,
  'createdAt': '2026-01-01T00:00:00Z',
  'updatedAt': '2026-01-02T00:00:00Z',
  'doneAt': status == 'done' ? '2026-01-03T00:00:00Z' : null,
  'meta': {
    'req': group ? null : {'url': 'https://example.com/$id'},
    'opts': {'path': directory, 'name': group ? 'folder' : name, 'size': size},
    if (!group && size > 0)
      'res': {
        'size': size,
        'files': [
          {'name': name, 'size': size},
        ],
      },
  },
  'progress': {'used': 0, 'speed': 10, 'downloaded': downloaded, 'uploadSpeed': 0, 'uploaded': 0},
});
