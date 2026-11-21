import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/window/app_window_bootstrap.dart';
import 'package:gopeed/core/window/app_window_payload.dart';
import 'package:gopeed/l10n/l10n.dart';

void main() {
  test('download progress payload round-trips only a task ID', () {
    final raw = AppWindowPayload.downloadProgress('task-123').toRaw();
    expect(jsonDecode(raw), {'type': 'download_progress', 'taskId': 'task-123'});
    final payload = AppWindowPayload.fromRaw(raw);
    expect(payload.type, AppWindowType.downloadProgress);
    expect(payload.taskId, 'task-123');
    expect(AppWindowBootstrap.subWindowTitle(payload.type, appLocalizationsFor('zh')), 'Gopeed - 下载进度弹窗');
  });

  test('invalid progress window task IDs are rejected', () {
    for (final taskId in [null, '', 42, <String>[]]) {
      final payload = AppWindowPayload.fromRaw(jsonEncode({'type': 'download_progress', 'taskId': taskId}));
      expect(payload.type, AppWindowType.main);
      expect(payload.taskId, isEmpty);
    }
  });

  test('create-task child window has a localized native title', () {
    expect(
      AppWindowBootstrap.subWindowTitle(AppWindowType.createTask, appLocalizationsFor('en')),
      'Gopeed - Create Task',
    );
    expect(AppWindowBootstrap.subWindowTitle(AppWindowType.createTask, appLocalizationsFor('zh')), 'Gopeed - 创建任务');
  });

  test('create-task payload only carries initial task data', () {
    final raw = AppWindowPayload.createTask(
      createTask: {
        'urls': ['https://example.com/file.zip'],
      },
    ).toRaw();
    final decoded = jsonDecode(raw) as Map<String, dynamic>;

    expect(decoded.keys, containsAll(<String>['type', 'createTask']));
    expect(decoded.keys, hasLength(2));
    expect(decoded, isNot(contains('apiToken')));
    expect(decoded, isNot(contains('theme')));
    expect(decoded, isNot(contains('createHistory')));
  });

  test('create-task payload preserves nested browser request parameters', () {
    final payload = AppWindowPayload.createTask(
      createTask: {
        'req': {
          'rawUrl': 'https://page.example/download',
          'url': 'https://cdn.example/archive.zip',
          'extra': {
            'method': 'GET',
            'header': {'Cookie': 'session=abc', 'Referer': 'https://page.example/', 'Authorization': 'Bearer token'},
            'body': '',
          },
          'labels': {'source': 'browser-extension'},
          'skipVerifyCert': true,
        },
        'opts': {'name': 'archive.zip', 'path': 'D:/Downloads', 'selectFiles': <int>[], 'extra': null},
      },
    );

    final decoded = AppWindowPayload.fromRaw(payload.toRaw()).createTask;

    expect(decoded?['req'], payload.createTask?['req']);
    expect(decoded?['opts'], payload.createTask?['opts']);
  });
}
