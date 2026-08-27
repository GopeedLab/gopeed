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
import 'package:gopeed/app/views/copy_button.dart';
import 'package:gopeed/app/views/task_speed_chart.dart';

class TestTaskDownloadingController extends TaskDownloadingController {
  @override
  void onInit() {
    isRunning.value = true;
  }

  @override
  getTasksState() async {}
}

class TestTaskDownloadedController extends TaskDownloadedController {
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
  String url = 'https://example.com/download',
  String path = '/downloads',
  Status status = Status.running,
}) {
  return Task(
    id: id,
    name: name,
    meta: Meta(
      req: Request(url: url),
      opts: Options(name: name, path: path),
    ),
    status: status,
    uploading: false,
    progress: Progress(
      used: 0,
      speed: 1024,
      downloaded: 2048,
      uploadSpeed: 0,
      uploaded: 0,
    ),
    createdAt: DateTime.now(),
    updatedAt: DateTime.now(),
  );
}

void main() {
  group('TaskDetail Speed Chart Integration Tests', () {
    late TaskController taskController;
    late TaskDownloadingController downloadingController;
    late TaskDownloadedController downloadedController;

    setUp(() {
      Get.reset();
      taskController = Get.put(TaskController());
      downloadingController = Get.put<TaskDownloadingController>(TestTaskDownloadingController());
      downloadedController = Get.put<TaskDownloadedController>(TestTaskDownloadedController());
    });

    tearDown(() {
      Get.reset();
    });

    testWidgets(
        '1. renders task detail view correctly with new speed chart section',
        (WidgetTester tester) async {
      final task1 = createTestTask(id: 'task-1', name: 'ubuntu.iso');
      taskController.selectTask.value = task1;
      downloadingController.speedHistory['task-1'] = [100.0, 250.0, 500.0];

      await tester.pumpWidget(
        const GetMaterialApp(
          home: TaskView(),
        ),
      );

      // Open the endDrawer to view task details
      taskController.scaffoldKey.currentState?.openEndDrawer();
      await tester.pumpAndSettle();

      // Verify TaskSpeedChart is present
      expect(find.byType(TaskSpeedChart), findsOneWidget);

      final chartWidget =
          tester.widget<TaskSpeedChart>(find.byType(TaskSpeedChart));
      expect(chartWidget.speedSamples, [100.0, 250.0, 500.0]);
    });

    testWidgets(
        '2. verifies all pre-existing elements and functionality are preserved',
        (WidgetTester tester) async {
      final task1 = createTestTask(
        id: 'task-1',
        name: 'flutter_package.tar.gz',
        url: 'https://example.com/flutter_package.tar.gz',
      );
      taskController.selectTask.value = task1;

      await tester.pumpWidget(
        const GetMaterialApp(
          home: TaskView(),
        ),
      );

      // Open end drawer
      taskController.scaffoldKey.currentState?.openEndDrawer();
      await tester.pumpAndSettle();

      // Check pre-existing UI elements:
      // 1. Task Name text
      expect(find.text('flutter_package.tar.gz'), findsWidgets);

      // 2. Task URL text & CopyButton
      expect(find.text('https://example.com/flutter_package.tar.gz'),
          findsWidgets);
      expect(find.byType(CopyButton), findsOneWidget);

      // 3. Folder open action icon
      expect(find.byIcon(Icons.folder_open), findsWidgets);

      // 4. Tab bar elements
      expect(find.byIcon(Icons.file_download), findsOneWidget);
      expect(find.byIcon(Icons.done), findsOneWidget);
    });

    testWidgets(
        '3. verifies chart displays data for the specific task being viewed and not another task',
        (WidgetTester tester) async {
      final taskA = createTestTask(id: 'task-A', name: 'fileA.zip');
      final taskB = createTestTask(id: 'task-B', name: 'fileB.zip');

      downloadingController.speedHistory['task-A'] = [10.0, 20.0, 30.0];
      downloadingController.speedHistory['task-B'] = [900.0, 950.0, 990.0];

      // Select Task A
      taskController.selectTask.value = taskA;

      await tester.pumpWidget(
        const GetMaterialApp(
          home: TaskView(),
        ),
      );

      taskController.scaffoldKey.currentState?.openEndDrawer();
      await tester.pumpAndSettle();

      var chart = tester.widget<TaskSpeedChart>(find.byType(TaskSpeedChart));
      expect(chart.speedSamples, [10.0, 20.0, 30.0]);
      expect(chart.speedSamples, isNot(contains(900.0)));

      // Switch selection to Task B
      taskController.selectTask.value = taskB;
      await tester.pumpAndSettle();

      chart = tester.widget<TaskSpeedChart>(find.byType(TaskSpeedChart));
      expect(chart.speedSamples, [900.0, 950.0, 990.0]);
      expect(chart.speedSamples, isNot(contains(10.0)));
    });
  });
}
