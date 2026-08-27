import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/meta.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/task.dart';
import 'package:gopeed/app/modules/task/controllers/task_list_controller.dart';

class TestTaskListController extends TaskListController {
  TestTaskListController()
      : super([
          Status.ready,
          Status.running,
          Status.pause,
          Status.wait,
          Status.error
        ], (a, b) => b.updatedAt.compareTo(a.updatedAt));
}

Task createMockTask({
  required String id,
  String name = 'test_file.zip',
  Status status = Status.running,
  int speed = 1024,
}) {
  return Task(
    id: id,
    name: name,
    meta: Meta(
      req: Request(url: 'https://example.com/$name'),
      opts: Options(name: name),
    ),
    status: status,
    uploading: false,
    progress: Progress(
      used: 0,
      speed: speed,
      downloaded: 1024,
      uploadSpeed: 0,
      uploaded: 0,
    ),
    createdAt: DateTime.now(),
    updatedAt: DateTime.now(),
  );
}

void main() {
  group('Task Rolling Speed History Buffer Tests', () {
    late TestTaskListController controller;

    setUp(() {
      controller = TestTaskListController();
    });

    test('1. correctly appends new speed samples as progress updates occur', () {
      final task1 = createMockTask(id: 'task-1', speed: 100);
      controller.updateSpeedHistory([task1]);
      expect(controller.getSpeedHistory('task-1'), [100.0]);

      final task2 = createMockTask(id: 'task-1', speed: 250);
      controller.updateSpeedHistory([task2]);
      expect(controller.getSpeedHistory('task-1'), [100.0, 250.0]);

      final task3 = createMockTask(id: 'task-1', speed: 500);
      controller.updateSpeedHistory([task3]);
      expect(controller.getSpeedHistory('task-1'), [100.0, 250.0, 500.0]);
    });

    test('2. correctly caps at the maximum length (60 samples) and drops the oldest sample', () {
      // Add 70 speed samples
      for (int i = 1; i <= 70; i++) {
        final task = createMockTask(id: 'task-cap', speed: i * 10);
        controller.updateSpeedHistory([task]);
      }

      final history = controller.getSpeedHistory('task-cap');
      expect(history.length, TaskListController.maxSpeedSamples);
      expect(history.length, 60);

      // The oldest 10 samples (10.0 to 100.0) should have been dropped.
      // Buffer should contain samples 11 to 70 (110.0 to 700.0).
      expect(history.first, 110.0);
      expect(history.last, 700.0);
    });

    test('3. verifies each task\'s buffer is independent of other tasks\' buffers', () {
      final taskA1 = createMockTask(id: 'task-A', speed: 100);
      final taskB1 = createMockTask(id: 'task-B', speed: 2000);
      controller.updateSpeedHistory([taskA1, taskB1]);

      final taskA2 = createMockTask(id: 'task-A', speed: 150);
      final taskB2 = createMockTask(id: 'task-B', speed: 2500);
      controller.updateSpeedHistory([taskA1, taskB1]);
      controller.updateSpeedHistory([taskA2, taskB2]);

      expect(controller.getSpeedHistory('task-A'), [100.0, 100.0, 150.0]);
      expect(controller.getSpeedHistory('task-B'), [2000.0, 2000.0, 2500.0]);
      expect(controller.getSpeedHistory('non-existent'), isEmpty);
    });

    test('4. stops appending once a task reaches terminal state (Done or Error)', () {
      final taskRunning = createMockTask(id: 'task-1', speed: 300, status: Status.running);
      controller.updateSpeedHistory([taskRunning]);
      expect(controller.getSpeedHistory('task-1'), [300.0]);

      // Task transitions to Done
      final taskDone = createMockTask(id: 'task-1', speed: 0, status: Status.done);
      controller.updateSpeedHistory([taskDone]);
      // Should not append 0.0 or any new sample when done
      expect(controller.getSpeedHistory('task-1'), [300.0]);

      // Task in Error state
      final taskError = createMockTask(id: 'task-2', speed: 500, status: Status.running);
      controller.updateSpeedHistory([taskError]);
      expect(controller.getSpeedHistory('task-2'), [500.0]);

      final taskErrorTerminal = createMockTask(id: 'task-2', speed: 0, status: Status.error);
      controller.updateSpeedHistory([taskErrorTerminal]);
      expect(controller.getSpeedHistory('task-2'), [500.0]);
    });

    test('5. clears buffer when task is deleted or removed from task list', () {
      final task1 = createMockTask(id: 'task-1', speed: 100);
      final task2 = createMockTask(id: 'task-2', speed: 200);
      controller.updateSpeedHistory([task1, task2]);

      expect(controller.getSpeedHistory('task-1'), [100.0]);
      expect(controller.getSpeedHistory('task-2'), [200.0]);

      // task-1 is deleted (omitted from the active task list)
      controller.updateSpeedHistory([task2]);
      expect(controller.getSpeedHistory('task-1'), isEmpty);
      expect(controller.getSpeedHistory('task-2'), [200.0, 200.0]);

      // Explicit clear method
      controller.clearSpeedHistory('task-2');
      expect(controller.getSpeedHistory('task-2'), isEmpty);
    });
  });
}
