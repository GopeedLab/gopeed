import 'package:flutter/widgets.dart';
import 'package:flutter/foundation.dart' show TargetPlatform, debugDefaultTargetPlatformOverride;
import 'package:gopeed/features/home/presentation/pages/home_page.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_file_manager.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gopeed/api/model/task.dart' as api;
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_file_browser_dialog.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_card/task_card.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:flutter/material.dart' show Icons;

api.Task raw(
  String id, {
  String parent = '',
  String status = 'running',
  String directory = '/downloads/folder',
  String name = 'file.txt',
}) => api.Task.fromJson({
  'id': id,
  'name': name,
  if (parent.isNotEmpty)
    'group': {
      'id': parent,
      'name': 'folder',
      'path': directory.contains('\\') ? r'C:\Downloads\folder' : '/downloads/folder',
      'createdAt': '2026-01-01T00:00:00Z',
    },
  'parentId': parent,
  'protocol': 'http',
  'status': status,
  'uploading': false,
  'createdAt': '2026-01-01T00:00:00Z',
  'updatedAt': '2026-01-01T00:00:00Z',
  'meta': {
    'req': {'url': 'https://example.com/$id'},
    'opts': {'path': directory, 'name': name, 'size': 100},
    'res': {
      'size': 100,
      'files': [
        {'name': name, 'size': 100},
      ],
    },
  },
  'progress': {'used': 0, 'speed': 10, 'downloaded': 25, 'uploaded': 0, 'uploadSpeed': 0},
});
TaskRecord grouped({String name = 'first.txt'}) =>
    taskRecordsFromApi([raw('child', parent: 'group', name: name)]).single;

class UpdatingTasks extends TasksController {
  @override
  Future<TasksState> build() async => TasksState(tasks: [grouped()]);
  void rename() => state = AsyncData(TasksState(tasks: [grouped(name: 'first (1).txt')]));
}

void main() {
  for (final platform in [TargetPlatform.android, TargetPlatform.iOS]) {
    testWidgets('tablet group expands its member tasks on $platform', (tester) async {
      debugDefaultTargetPlatformOverride = platform;
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      tester.view.physicalSize = const Size(1024, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [tasksControllerProvider.overrideWith(UpdatingTasks.new)],
          child: shad.ShadcnApp(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            home: const HomePage(),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));
      await tester.tap(find.text('folder').first);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));
      expect(find.byType(TaskFileManagerView), findsNothing);
      expect(find.byType(TaskCard), findsNWidgets(2));
      expect(find.text('first.txt'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      debugDefaultTargetPlatformOverride = null;
    });
  }
  test('Windows backend root and child path are rebased on a POSIX client', () {
    final record = taskRecordsFromApi([raw('child', parent: 'group', directory: r'C:\Downloads\folder\nested')]).single;
    expect(record.files.single.path, 'nested');
    expect(record.files.single.name, 'file.txt');
  });
  testWidgets('open group file browser updates renamed child from task polling', (tester) async {
    tester.view.physicalSize = const Size(1100, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    late UpdatingTasks controller;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [tasksControllerProvider.overrideWith(() => controller = UpdatingTasks())],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Consumer(
            builder: (context, ref, _) {
              final tasks = ref.watch(tasksControllerProvider).value?.tasks;
              return shad.PrimaryButton(
                onPressed: tasks == null ? null : () => showTaskFileBrowserDialog(context, tasks.single),
                child: const Text('Open'),
              );
            },
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();
    expect(find.text('first.txt'), findsOneWidget);
    controller.rename();
    await tester.pumpAndSettle();
    expect(find.text('first (1).txt'), findsOneWidget);
  });
  for (final width in [390.0, 700.0, 1024.0]) {
    testWidgets('a running group with paused members offers resume on its card at $width', (tester) async {
      tester.view.physicalSize = Size(width, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final record = taskRecordsFromApi([
        raw('running', parent: 'group'),
        raw('paused', parent: 'group', status: 'pause'),
      ]).single;
      expect(record.canResume, isTrue);
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Center(
            child: TaskCard(
              task: record,
              selected: false,
              batchMode: false,
              selectedInBatch: false,
              onPressed: () {},
              onToggleBatch: () {},
              onPause: () {},
              onResume: () {},
              onDelete: () {},
            ),
          ),
        ),
      );
      await tester.pump();
      expect(find.byIcon(Icons.play_arrow), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
    });
  }
}
