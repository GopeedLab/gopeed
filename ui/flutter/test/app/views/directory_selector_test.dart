import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/app/views/directory_selector.dart';

void main() {
  test('updateDirectorySelection updates the controller and notifies', () {
    final controller = TextEditingController(text: '/old/path');
    var notified = 0;

    updateDirectorySelection(controller, '/new/path',
        onChanged: () => notified++);

    expect(controller.text, '/new/path');
    expect(controller.selection.baseOffset, '/new/path'.length);
    expect(notified, 1);
  });

  test('updateDirectorySelection skips duplicate values', () {
    final controller = TextEditingController(text: '/same/path');
    var notified = 0;

    updateDirectorySelection(controller, '/same/path',
        onChanged: () => notified++);

    expect(controller.text, '/same/path');
    expect(notified, 0);
  });
}
