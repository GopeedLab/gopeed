import 'dart:async';
import 'dart:io';

import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/tasks/application/task_stats_provider.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/pages/download_progress_window_page.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_progress_bar.dart';
import 'package:gopeed/l10n/l10n.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/theme/app_design_tokens.dart';

void main() {
  Future<_Tasks> mount(
    WidgetTester tester, {
    Size size = AppDesignTokens.downloadProgressWindowSize,
    VoidCallback? onClose,
    bool dark = false,
    bool modal = true,
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final container = ProviderContainer(
      overrides: [
        tasksControllerProvider.overrideWith(_Tasks.new),
        taskStatsProvider.overrideWith((ref, request) async => null),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: ShadcnApp(
          theme: dark ? AppTheme.dark() : AppTheme.light(),
          materialTheme: dark ? AppTheme.materialDark() : AppTheme.materialLight(),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: DownloadProgressWindowPage(taskId: 'download', onClose: onClose ?? () {}, modal: modal),
        ),
      ),
    );
    await tester.pump();
    return container.read(tasksControllerProvider.notifier) as _Tasks;
  }

  for (final width in [320.0, 768.0, 1024.0]) {
    for (final dark in [false, true]) {
      testWidgets('download window adapts to width $width, dark=$dark', (tester) async {
        await mount(tester, size: Size(width, AppDesignTokens.downloadProgressWindowSize.height), dark: dark);
        expect(find.text('25.0%'), findsOneWidget);
        expect(find.text('128 KB/s'), findsWidgets);
        expect(find.text('64 KB/s'), findsWidgets);
        expect(find.byKey(const ValueKey('download-speed-chart')), findsOneWidget);
        expect(find.text('1 MB / 4 MB'), findsOneWidget);
        expect(find.text('30 seconds remaining'), findsOneWidget);
        expect(find.byKey(const ValueKey('download-progress-url')), findsNothing);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
      });
    }
  }

  testWidgets('pause, resume, completion and close follow live task state', (tester) async {
    var closed = false;
    final tasks = await mount(tester, onClose: () => closed = true);
    await tester.tap(find.byKey(const ValueKey('download-progress-pause')));
    await tester.pump();
    expect(tasks.actions, ['pause:download']);
    expect(find.byKey(const ValueKey('download-progress-resume')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('download-progress-resume')));
    await tester.pump();
    expect(tasks.actions, ['pause:download', 'resume:download']);
    tasks.emit(TaskStatus.completed);
    await tester.pump();
    expect(find.text('100.0%'), findsOneWidget);
    expect(find.byKey(const ValueKey('download-progress-pause')), findsNothing);
    expect(find.byKey(const ValueKey('download-progress-resume')), findsNothing);
    await tester.tap(find.byKey(const ValueKey('download-progress-close')));
    await tester.pump();
    expect(closed, isTrue);
    expect(tasks.actions, hasLength(2));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('background remains available while pause is pending', (tester) async {
    var closed = false;
    final tasks = await mount(tester, onClose: () => closed = true);
    tasks.pauseRequest = Completer<void>();
    await tester.tap(find.byKey(const ValueKey('download-progress-pause')));
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('download-progress-close')));
    await tester.pump();
    expect(closed, isTrue);
    tasks.pauseRequest!.complete();
    await tester.pump();
    expect(find.byKey(const ValueKey('download-progress-resume')), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('failed pause restores controls and closing errors are reported', (tester) async {
    final tasks = await mount(tester, onClose: () => throw StateError('close failed'));
    tasks.pauseRequest = Completer<void>();
    await tester.tap(find.byKey(const ValueKey('download-progress-pause')));
    await tester.pump();
    tasks.pauseRequest!.completeError(StateError('pause failed'));
    await tester.pump();
    expect(find.textContaining('pause failed'), findsOneWidget);
    tasks.pauseRequest = null;
    await tester.tap(find.byKey(const ValueKey('download-progress-pause')));
    await tester.pump();
    expect(find.byKey(const ValueKey('download-progress-resume')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('download-progress-close')));
    await tester.pump();
    expect(find.textContaining('close failed'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 6));
  });

  testWidgets('unknown size, download error, polling error and deletion are visible', (tester) async {
    final tasks = await mount(tester);
    tasks.emit(TaskStatus.downloading, unknownSize: true);
    await tester.pump();
    expect(tester.widget<TaskProgressBar>(find.byType(TaskProgressBar)).indeterminate, isTrue);
    expect(find.text('25.0%'), findsNothing);
    tasks.emit(TaskStatus.failed, error: 'Connection lost');
    await tester.pump();
    expect(find.text('Connection lost'), findsOneWidget);
    expect(find.byKey(const ValueKey('download-progress-resume')), findsOneWidget);
    tasks.pollFailure();
    await tester.pump();
    expect(find.textContaining('offline'), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);
    tasks.remove();
    await tester.pump();
    expect(find.text('Task not found'), findsOneWidget);
    expect(find.byKey(const ValueKey('download-progress-pause')), findsNothing);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('closing an active download does not pause it', (tester) async {
    var closed = false;
    final tasks = await mount(tester, onClose: () => closed = true);
    await tester.tap(find.byKey(const ValueKey('download-progress-close')));
    await tester.pump();
    expect(closed, isTrue);
    expect(tasks.actions, isEmpty);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('connection details expand on demand without hiding the footer', (tester) async {
    await mount(tester);
    final details = find.byKey(const ValueKey('download-progress-details'));
    await tester.ensureVisible(details);
    await tester.tap(details);
    await tester.pump();
    expect(find.byKey(const ValueKey('download-progress-path')), findsOneWidget);
    expect(find.byKey(const ValueKey('download-progress-close')).hitTestable(), findsOneWidget);
    await tester.tap(details);
    await tester.pump();
    expect(find.byKey(const ValueKey('download-progress-path')), findsNothing);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  for (final deleteFiles in [false, true]) {
    testWidgets('cancel confirms file deletion=$deleteFiles before closing', (tester) async {
      var closed = false;
      final tasks = await mount(tester, onClose: () => closed = true);
      await tester.tap(find.byKey(const ValueKey('download-progress-cancel')));
      await tester.pump(const Duration(milliseconds: 300));
      expect(tasks.actions, isEmpty);
      if (deleteFiles) {
        await tester.tap(find.byKey(const ValueKey('task-delete-files-switch')));
        await tester.pump();
      }
      await tester.tap(find.byType(DestructiveButton));
      await tester.pump(const Duration(milliseconds: 300));
      expect(tasks.actions, ['delete:download:$deleteFiles']);
      expect(closed, isTrue);
      await tester.pumpWidget(const SizedBox());
    });
  }

  testWidgets('dismissing cancellation keeps the task and window', (tester) async {
    var closed = false;
    final tasks = await mount(tester, onClose: () => closed = true);
    await tester.tap(find.byKey(const ValueKey('download-progress-cancel')));
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byType(SecondaryButton));
    await tester.pump(const Duration(milliseconds: 300));
    expect(tasks.actions, isEmpty);
    expect(closed, isFalse);
    await tester.pumpWidget(const SizedBox());
  });

  for (final width in [320.0, 520.0]) {
    testWidgets(
      'completed file actions stay visible at compact width $width',
      (tester) async {
        final tasks = await mount(tester, size: Size(width, 440), modal: false);
        final openFile = find.byKey(const ValueKey('download-progress-open-file'));
        final openFolder = find.byKey(const ValueKey('download-progress-open-folder'));
        expect(openFile, findsNothing);
        expect(openFolder, findsNothing);
        tasks.emit(TaskStatus.completed);
        await tester.pump();
        expect(openFile.hitTestable(), findsOneWidget);
        expect(openFolder.hitTestable(), findsOneWidget);
        expect(find.byKey(const ValueKey('download-progress-close')).hitTestable(), findsOneWidget);
        expect(tester.takeException(), isNull);
        for (final group in [false, true]) {
          tasks.emit(TaskStatus.completed, isFolder: !group, isGroup: group);
          await tester.pump();
          expect(openFile, findsNothing);
          expect(openFolder.hitTestable(), findsOneWidget);
        }
        await tester.pumpWidget(const SizedBox());
      },
      variant: TargetPlatformVariant({TargetPlatform.windows, TargetPlatform.macOS, TargetPlatform.linux}),
    );
  }

  for (final reveal in [false, true]) {
    testWidgets(
      'completed ${reveal ? 'folder' : 'file'} action uses task path and reports missing files',
      (tester) async {
        final tasks = await mount(tester);
        tasks.emit(TaskStatus.completed);
        await tester.pump();
        final checkedPaths = <String>[];
        await IOOverrides.runZoned(
          () async {
            await tester.tap(
              find.byKey(ValueKey(reveal ? 'download-progress-open-folder' : 'download-progress-open-file')),
            );
            await tester.pump();
            await tester.pump(const Duration(milliseconds: 300));
            expect(checkedPaths.first, '/downloads/archive.zip');
            final l10n = appLocalizationsFor('en');
            expect(
              find.text(
                reveal
                    ? l10n.unableLocatePath('/downloads/archive.zip')
                    : l10n.unableOpenPath('/downloads/archive.zip'),
              ),
              findsOneWidget,
            );
            expect(tasks.actions, isEmpty);
          },
          fseGetType: (path, followLinks) async {
            checkedPaths.add(path);
            return FileSystemEntityType.notFound;
          },
        );
        await tester.pumpWidget(const SizedBox());
        await tester.pump(const Duration(seconds: 6));
      },
      variant: TargetPlatformVariant({TargetPlatform.windows}),
    );
  }
}

class _Tasks extends TasksController {
  final actions = <String>[];
  Completer<void>? pauseRequest;
  @override
  Future<void> deleteSelected(List<String>? ids, {bool force = false}) async {
    actions.add('delete:${ids!.single}:$force');
    remove();
  }

  @override
  Future<TasksState> build() async => TasksState(tasks: [_task(TaskStatus.downloading)]);
  void emit(
    TaskStatus status, {
    bool unknownSize = false,
    String? error,
    bool isFolder = false,
    bool isGroup = false,
  }) => state = AsyncData(
    TasksState(
      tasks: [_task(status, unknownSize: unknownSize, error: error, isFolder: isFolder, isGroup: isGroup)],
    ),
  );
  void remove() => state = const AsyncData(TasksState(tasks: []));
  void pollFailure() => state = AsyncData(state.requireValue.copyWith(lastError: StateError('offline')));
  @override
  Future<void> pause(String id) async {
    actions.add('pause:$id');
    await pauseRequest?.future;
    emit(TaskStatus.paused);
  }

  @override
  Future<void> resume(String id) async {
    actions.add('resume:$id');
    emit(TaskStatus.downloading);
  }
}

TaskRecord _task(
  TaskStatus status, {
  bool unknownSize = false,
  String? error,
  bool isFolder = false,
  bool isGroup = false,
}) => TaskRecord(
  id: 'download',
  name: 'archive.zip',
  status: status,
  isFolder: isFolder,
  isGroup: isGroup,
  downloaded: '1 MB',
  total: unknownSize ? null : '4 MB',
  progress: unknownSize ? null : 0.25,
  speed: '128 KB/s',
  speedBytes: 131072,
  downloadedBytes: 1048576,
  downloadDuration: const Duration(seconds: 16),
  remainingSeconds: 30,
  error: error,
  url: 'https://example.com/archive.zip',
  storagePath: '/downloads/archive.zip',
  files: const [],
  uploading: false,
);
