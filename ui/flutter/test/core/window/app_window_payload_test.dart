import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/window/app_window_payload.dart';

void main() {
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
}
