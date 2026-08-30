import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/common/task_event.dart';

void main() {
  test('task event mask combines subscribed event types', () {
    expect({TaskEventType.done}.mask, 1);
    expect({TaskEventType.error}.mask, 2);
    expect({TaskEventType.done, TaskEventType.error}.mask, 3);
  });

  test('task event decodes terminal payload', () {
    final event = TaskEvent.fromJson({
      'type': 'task.error',
      'taskId': 'task-1',
      'name': 'archive.zip',
      'error': 'network error',
    });

    expect(event.type, TaskEventType.error);
    expect(event.taskId, 'task-1');
    expect(event.name, 'archive.zip');
    expect(event.error, 'network error');
  });
}
