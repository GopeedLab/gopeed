import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/task.dart' as api;
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_file_tree.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_progress_bar.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final scenario in [
    (protocol: api.Protocol.http, completed: false),
    (protocol: api.Protocol.http, completed: true),
    (protocol: api.Protocol.bt, completed: false),
  ]) {
    testWidgets('file progress stays bounded and respects completion state: $scenario', (tester) async {
      tester.view.physicalSize = const Size(520, 420);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final total = scenario.completed ? 10 : 3;
      final task = TaskRecord(
        id: 'estimated-size',
        name: 'video.mp4',
        protocol: scenario.protocol,
        status: scenario.completed ? TaskStatus.completed : TaskStatus.downloading,
        downloaded: '10 B',
        total: '$total B',
        progress: 1,
        url: 'https://example.com/video',
        storagePath: '/downloads/video.mp4',
        files: [TaskFileNode(path: '', name: 'video.mp4', sizeBytes: total, resourceIndex: 0)],
        uploading: false,
      );
      final runtime = api.TaskRuntimeStatus(
        status: scenario.completed ? api.Status.done : api.Status.running,
        used: 0,
        speed: 2,
        downloaded: 10,
        total: total,
        uploadSpeed: 0,
        uploaded: 0,
        files: [api.FileRuntimeStatus(index: 0, size: total, downloaded: 10)],
      );
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Padding(
            padding: const EdgeInsets.all(24),
            child: TaskFileTree(task: task, runtimeStatus: runtime),
          ),
        ),
      );
      await tester.pump();
      final progress = tester.widget<TaskProgressBar>(find.byKey(const ValueKey('task-files-total-progress')));
      expect(progress.value, 1);
      expect(find.text('100%'), findsOneWidget);
      expect(find.text('333%'), findsNothing);
      final file = find.byKey(const ValueKey('task-file-progress-file:0'));
      final check = find.descendant(of: file, matching: find.byIcon(shad.BootstrapIcons.checkCircleFill));
      final ring = find.descendant(of: file, matching: find.byType(shad.CircularProgressIndicator));
      if (!scenario.completed) {
        expect(check, findsNothing);
        expect(ring, findsOneWidget);
        expect(tester.widget<shad.CircularProgressIndicator>(ring).value, 1);
      } else {
        expect(check, findsOneWidget);
        expect(ring, findsNothing);
      }
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    });
  }
}
