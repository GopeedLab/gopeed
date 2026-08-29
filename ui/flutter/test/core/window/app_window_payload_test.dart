import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/window/app_window_bootstrap.dart';
import 'package:gopeed/core/window/app_window_payload.dart';
import 'package:gopeed/l10n/l10n.dart';

void main() {
  test('create-task child window has a localized native title', () {
    expect(
      AppWindowBootstrap.subWindowTitle(AppWindowType.createTask, appLocalizationsFor('en')),
      'Create Task - Gopeed',
    );
    expect(AppWindowBootstrap.subWindowTitle(AppWindowType.createTask, appLocalizationsFor('zh')), '创建任务 - Gopeed');
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
