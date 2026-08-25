import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:get/get.dart' hide Progress;
import 'package:gopeed/api/model/meta.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/task.dart';
import 'package:gopeed/app/modules/task/controllers/task_controller.dart';
import 'package:gopeed/app/modules/task/controllers/task_downloaded_controller.dart';
import 'package:gopeed/app/modules/task/controllers/task_downloading_controller.dart';
import 'package:gopeed/app/modules/task/views/task_view.dart';
import 'package:gopeed/app/views/task_speed_chart.dart';

class MockTaskDownloadingController extends TaskDownloadingController {
  @override
  void onInit() {
    isRunning.value = true;
  }

  @override
  getTasksState() async {}
}

class MockTaskDownloadedController extends TaskDownloadedController {
  @override
  void onInit() {
    isRunning.value = true;
  }

  @override
  getTasksState() async {}
}

Task createTestTask({
  required String id,
  required String name,
  Status status = Status.running,
  int speed = 1024,
}) {
  return Task(
    id: id,
    name: name,
    meta: Meta(
      req: Request(url: 'https://example.com/$name'),
      opts: Options(name: name, path: '/downloads'),
    ),
    status: status,
    uploading: false,
    progress: Progress(
      used: 0,
      speed: speed,
      downloaded: 4096,
      uploadSpeed: 0,
      uploaded: 0,
    ),
    createdAt: DateTime.now(),
    updatedAt: DateTime.now(),
  );
}

void main() {
  group('TaskSpeedChart Edge Case Tests', () {
    testWidgets('1. Edge Case 1: handles < 2 samples without error or crash',
        (WidgetTester tester) async {
      // Test 0 samples
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: TaskSpeedChart(
              speedSamples: [],
              height: 80,
            ),
          ),
        ),
      );
      expect(find.byType(TaskSpeedChart), findsOneWidget);
      expect(find.byType(CustomPaint), findsWidgets);
      expect(tester.takeException(), isNull);

      // Test 1 sample
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: TaskSpeedChart(
              speedSamples: [500000.0],
              height: 80,
            ),
          ),
        ),
      );
      expect(find.byType(TaskSpeedChart), findsOneWidget);
      expect(find.byType(CustomPaint), findsWidgets);
      expect(tester.takeException(), isNull);
    });

    testWidgets('2. Edge Case 2: paused task visually dims line while preserving data',
        (WidgetTester tester) async {
      final samples = [100.0, 200.0, 300.0];

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: TaskSpeedChart(
              speedSamples: samples,
              isPaused: true,
              height: 80,
            ),
          ),
        ),
      );

      final chart = tester.widget<TaskSpeedChart>(find.byType(TaskSpeedChart));
      expect(chart.isPaused, isTrue);
      expect(chart.speedSamples, samples);
      expect(tester.takeException(), isNull);
    });

    testWidgets('3. Edge Case 3: completed task displays settled history or is gracefully handled',
        (WidgetTester tester) async {
      final samples = [500.0, 1000.0, 1500.0];

      // Completed with history
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: TaskSpeedChart(
              speedSamples: samples,
              isCompleted: true,
              height: 80,
            ),
          ),
        ),
      );

      final chart = tester.widget<TaskSpeedChart>(find.byType(TaskSpeedChart));
      expect(chart.isCompleted, isTrue);
      expect(chart.speedSamples, samples);
      expect(tester.takeException(), isNull);
    });

    testWidgets('4. Edge Case 4: throttles visual repaints on high-frequency data changes',
        (WidgetTester tester) async {
      List<double> dynamicSamples = [10.0, 20.0];

      await tester.pumpWidget(
        StatefulBuilder(
          builder: (context, setState) {
            return MaterialApp(
              home: Scaffold(
                body: TaskSpeedChart(
                  speedSamples: dynamicSamples,
                  throttleDuration: const Duration(milliseconds: 300),
                  height: 80,
                ),
              ),
            );
          },
        ),
      );

      // Rapidly fire 20 data updates in quick succession
      for (int i = 1; i <= 20; i++) {
        dynamicSamples = [10.0 * i, 20.0 * i, 30.0 * i];
        await tester.pumpWidget(
          MaterialApp(
            home: Scaffold(
              body: TaskSpeedChart(
                speedSamples: dynamicSamples,
                throttleDuration: const Duration(milliseconds: 300),
                height: 80,
              ),
            ),
          ),
        );
        // Step forward by 10ms (less than the 300ms throttle)
        await tester.pump(const Duration(milliseconds: 10));
      }

      expect(tester.takeException(), isNull);

      // Advance clock past the throttle window to allow final settled frame
      await tester.pumpAndSettle(const Duration(milliseconds: 400));
      expect(tester.takeException(), isNull);
    });
  });

  group('TaskDetail View Completed and Paused State Tests', () {
    late TaskController taskController;
    late TaskDownloadingController downloadingController;
    late TaskDownloadedController downloadedController;

    setUp(() {
      Get.reset();
      taskController = Get.put(TaskController());
      downloadingController =
          Get.put<TaskDownloadingController>(MockTaskDownloadingController());
      downloadedController =
          Get.put<TaskDownloadedController>(MockTaskDownloadedController());
    });

    tearDown(() {
      Get.reset();
    });

    testWidgets('5. TaskView drawer passes isPaused=true when task is paused',
        (WidgetTester tester) async {
      final pausedTask =
          createTestTask(id: 'pause-task', name: 'big_game.iso', status: Status.pause);
      taskController.selectTask.value = pausedTask;
      downloadingController.speedHistory['pause-task'] = [500.0, 800.0];

      await tester.pumpWidget(
        const GetMaterialApp(
          home: TaskView(),
        ),
      );

      taskController.scaffoldKey.currentState?.openEndDrawer();
      await tester.pumpAndSettle();

      final chart = tester.widget<TaskSpeedChart>(find.byType(TaskSpeedChart));
      expect(chart.isPaused, isTrue);
      expect(chart.speedSamples, [500.0, 800.0]);
    });

    testWidgets('6. TaskView drawer renders chart with isCompleted=true for completed task',
        (WidgetTester tester) async {
      final doneTask =
          createTestTask(id: 'done-task', name: 'completed.zip', status: Status.done);
      taskController.selectTask.value = doneTask;

      await tester.pumpWidget(
        const GetMaterialApp(
          home: TaskView(),
        ),
      );

      taskController.scaffoldKey.currentState?.openEndDrawer();
      await tester.pumpAndSettle();

      expect(find.byType(TaskSpeedChart), findsOneWidget);
      final chart = tester.widget<TaskSpeedChart>(find.byType(TaskSpeedChart));
      expect(chart.isCompleted, isTrue);
      expect(find.text('completed.zip'), findsWidgets);
    });
  });
}
