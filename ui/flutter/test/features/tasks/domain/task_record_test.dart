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
