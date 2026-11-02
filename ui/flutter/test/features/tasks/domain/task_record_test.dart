import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/task.dart' as api;
import 'package:gopeed/core/icons/gopeed_icons.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';

void main() {
  api.Task buildTask(List<int> selection, int size, {String? doneAt, String protocol = 'bt'}) => api.Task.fromJson({
    'id': 'bt-selection',
    'name': 'bundle',
    'protocol': protocol,
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

  test('members aggregate through parentId without a parent task', () {
    final records = taskRecordsFromApi([
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
    ]);
    expect(records.map((record) => record.id), ['group', 'standalone']);
    final record = records.first;
    expect(record.isGroup, isTrue);
    expect(record.members.length, 2);
    expect(record.totalBytes, 400);
    expect(record.downloadedBytes, 100);
    expect(record.speedBytes, 20);
    expect(record.progress, 0.25);
    expect(record.storagePath, '/downloads/folder');
    expect(record.files.map((file) => file.path), ['', 'nested']);
    expect(record.files.last.name, 'movie.mp4');
    expect(record.matchesSearch('second'), isTrue);
    expect(record.canUpdateUrl, isFalse);
  });

  test('group preserves uploading while member upload speed is momentarily zero', () {
    final member = groupTestTask('uploading', parentId: 'group');
    member.uploading = true;
    member.progress.uploadSpeed = 0;
    final group = taskRecordsFromApi([member]).single;
    expect(group.uploading, isTrue);
    expect(group.uploadSpeedBytes, 0);
  });

  for (final scenario in <(List<String>, TaskStatus, bool)>[
    (['running', 'error', 'pause'], TaskStatus.downloading, false),
    (['error', 'pause', 'wait'], TaskStatus.failed, false),
    (['pause', 'wait', 'done'], TaskStatus.paused, false),
    (['done', 'done'], TaskStatus.completed, false),
    (['wait', 'done'], TaskStatus.downloading, true),
  ]) {
    test('group status priority for ${scenario.$1}', () {
      final record = taskRecordsFromApi([
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
    final unknown = groupTestTask('unknown', parentId: 'group', size: 0);
    final record = taskRecordsFromApi([unknown, groupTestTask('known', parentId: 'group')]).single;
    expect(record.totalBytes, isNull);
    expect(record.progress, isNull);
    unknown.status = api.Status.done;
    final empty = taskRecordsFromApi([unknown]).single;
    expect(empty.totalBytes, 0);
    expect(empty.progress, 1);
  });

  test('Windows group directories retain their relative file tree', () {
    final records = taskRecordsFromApi([
      groupTestTask('child', parentId: 'group', directory: r'C:\Downloads\folder\nested'),
    ]);
    expect(records.single.files.single.path, 'nested');
    expect(records.single.files.single.name, 'file.txt');
  });

  test('folder member file progress spreads the member ratio over its files', () {
    final member = api.Task.fromJson({
      'id': 'pack',
      'name': 'pack',
      'parentId': 'group',
      'group': {'id': 'group', 'name': 'folder', 'path': '/downloads/folder', 'createdAt': '2026-01-01T00:00:00Z'},
      'protocol': 'http',
      'status': 'running',
      'uploading': false,
      'createdAt': '2026-01-01T00:00:00Z',
      'updatedAt': '2026-01-02T00:00:00Z',
      'doneAt': null,
      'meta': {
        'req': {'url': 'https://example.com/pack'},
        'opts': {'path': '/downloads/folder/pack', 'name': 'pack', 'size': 30},
        'res': {
          'name': 'pack',
          'size': 30,
          'files': [
            {'name': 'a.bin', 'size': 10},
            {'name': 'b.bin', 'size': 20},
          ],
        },
      },
      'progress': {'used': 0, 'speed': 5, 'downloaded': 15, 'uploadSpeed': 0, 'uploaded': 0},
    });
    final record = taskRecordsFromApi([member]).single;
    expect(record.files.map((file) => file.name), ['a.bin', 'b.bin']);
    expect(record.files.first.downloadedBytes, 5);
    expect(record.files.last.downloadedBytes, 10);
  });

  test('members outside the group root render under their own folder name', () {
    final folderMember = api.Task.fromJson({
      'id': 'pack',
      'name': 'pack',
      'parentId': 'group',
      'group': {'id': 'group', 'name': 'folder', 'path': '/downloads/folder', 'createdAt': '2026-01-01T00:00:00Z'},
      'protocol': 'http',
      'status': 'running',
      'uploading': false,
      'createdAt': '2026-01-01T00:00:00Z',
      'updatedAt': '2026-01-02T00:00:00Z',
      'doneAt': null,
      'meta': {
        'req': {'url': 'https://example.com/pack'},
        'opts': {'path': '/elsewhere/pack', 'name': 'pack', 'size': 30},
        'res': {
          'name': 'pack',
          'size': 30,
          'files': [
            {'name': 'a.bin', 'size': 30},
          ],
        },
      },
      'progress': {'used': 0, 'speed': 5, 'downloaded': 15, 'uploadSpeed': 0, 'uploaded': 0},
    });
    final record = taskRecordsFromApi([
      groupTestTask('loose', parentId: 'group', directory: '/elsewhere'),
      folderMember,
    ]).single;
    for (final file in record.files) {
      expect(file.path.startsWith('..'), isFalse);
    }
    expect(record.files.first.path, '');
    expect(record.files.first.name, 'file.txt');
    expect(record.files.last.path, 'pack');
    expect(record.files.last.name, 'a.bin');
  });

  test('completed group reports the wall span from creation to finish', () {
    final record = taskRecordsFromApi([groupTestTask('child', parentId: 'group', status: 'done')]).single;
    expect(record.downloadDuration, const Duration(days: 2));
  });

  for (final completed in [false, true]) {
    test('group timestamps use chronological maxima when completed=$completed', () {
      final first = groupTestTask('first', parentId: 'group', status: completed ? 'done' : 'running')
        ..updatedAt = DateTime.utc(2026, 1, 6)
        ..doneAt = DateTime.utc(2026, 1, 5);
      final second = groupTestTask('second', parentId: 'group', status: 'done')
        ..updatedAt = DateTime.utc(2026, 1, 3)
        ..doneAt = DateTime.utc(2026, 1, 4);
      final legacy = groupTestTask('legacy', parentId: 'group', status: 'done')..doneAt = null;
      final record = taskRecordsFromApi([first, second, legacy]).single;
      expect(record.updatedAt, DateTime.utc(2026, 1, 6));
      expect(record.doneAt, completed ? DateTime.utc(2026, 1, 5) : null);
      expect(record.downloadDuration, completed ? const Duration(days: 4) : null);
    });
  }

  test('group creation bounds updated time and legacy completion stays unknown', () {
    final member = groupTestTask('legacy', parentId: 'group', status: 'done')..doneAt = null;
    member.group = api.TaskGroup(
      id: 'group',
      name: 'folder',
      path: '/downloads/folder',
      createdAt: DateTime.utc(2026, 1, 4),
    );
    final record = taskRecordsFromApi([member]).single;
    expect(record.updatedAt, member.group!.createdAt);
    expect(record.doneAt, isNull);
    expect(record.downloadDuration, isNull);
  });

  test('completed group reports the members average speed instead of 0 B/s', () {
    final record = taskRecordsFromApi([
      groupTestTask(
        'first',
        parentId: 'group',
        status: 'done',
        size: 100,
        downloaded: 100,
        usedNanoseconds: 10 * 1000000000,
      ),
      groupTestTask(
        'second',
        parentId: 'group',
        status: 'done',
        size: 50,
        downloaded: 50,
        usedNanoseconds: 5 * 1000000000,
      ),
    ]).single;
    // 150 bytes over 15 combined active seconds, not the 0 of the idle aggregate.
    expect(record.speedBytes, 10);
    expect(record.speed, '10 B/s');
  });

  test('paused group keeps the idle aggregate instead of an average speed', () {
    final record = taskRecordsFromApi([
      groupTestTask('first', parentId: 'group', status: 'pause', downloaded: 100, usedNanoseconds: 10 * 1000000000),
    ]).single;
    expect(record.speedBytes, 0);
  });

  test('selected files retain original indexes for runtime progress', () {
    final record = TaskRecord.fromApi(buildTask([2, 0], 40));
    expect(record.files.map((file) => file.name), ['a.txt', 'c.txt']);
    expect(record.files.map((file) => file.resourceIndex), [0, 2]);
    expect(record.files.last.path, 'nested');
    expect(record.totalBytes, 40);
    expect(record.progress, 0.25);
  });

  test('tasks with the ftp protocol decode', () {
    final task = buildTask([], 60, protocol: 'ftp');
    expect(task.protocol, api.Protocol.ftp);
    final record = TaskRecord.fromApi(task);
    expect(record.protocol, api.Protocol.ftp);
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
  String status = 'running',
  int size = 100,
  int downloaded = 25,
  int usedNanoseconds = 0,
  String directory = '/downloads/folder',
  String name = 'file.txt',
}) => api.Task.fromJson({
  'id': id,
  'name': name,
  'parentId': parentId,
  if (parentId.isNotEmpty)
    'group': {
      'id': parentId,
      'name': 'folder',
      'path': directory.contains('\\') ? r'C:\Downloads\folder' : '/downloads/folder',
      'createdAt': '2026-01-01T00:00:00Z',
    },
  'protocol': 'http',
  'status': status,
  'uploading': false,
  'createdAt': '2026-01-01T00:00:00Z',
  'updatedAt': '2026-01-02T00:00:00Z',
  'doneAt': status == 'done' ? '2026-01-03T00:00:00Z' : null,
  'meta': {
    'req': {'url': 'https://example.com/$id'},
    'opts': {'path': directory, 'name': name, 'size': size},
    if (size > 0)
      'res': {
        'size': size,
        'files': [
          {'name': name, 'size': size},
        ],
      },
  },
  'progress': {'used': usedNanoseconds, 'speed': 10, 'downloaded': downloaded, 'uploadSpeed': 0, 'uploaded': 0},
});
