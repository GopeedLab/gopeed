import 'dart:async';
import 'dart:io';

import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/task_stats.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/tasks/application/task_stats_provider.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/pages/download_progress_window_page.dart';
import 'package:gopeed/features/tasks/presentation/widgets/download_progress_connections.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_progress_bar.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_statistics/task_statistics_tab.dart';
import 'package:gopeed/l10n/l10n.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/theme/app_design_tokens.dart';

void main() {
  Future<_Tasks> mount(
    WidgetTester tester, {
    Size size = AppDesignTokens.downloadProgressWindowSize,
    VoidCallback? onClose,
    bool dark = false,
    TaskStats? stats,
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final container = ProviderContainer(
      overrides: [
        tasksControllerProvider.overrideWith(_Tasks.new),
        taskStatsProvider.overrideWith((ref, request) async => stats),
      ],
    );
    addTearDown(container.dispose);
    if (onClose != null) {
      const channel = MethodChannel('window_manager');
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (call) async {
        if (call.method == 'close') onClose();
        return null;
      });
      addTearDown(() => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, null));
    }
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: ShadcnApp(
          theme: dark ? AppTheme.dark() : AppTheme.light(),
          materialTheme: dark ? AppTheme.materialDark() : AppTheme.materialLight(),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const DownloadProgressWindowPage(taskId: 'download'),
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
        expect(find.text('25.0%'), findsNothing);
        final progress = tester.widget<TaskProgressBar>(find.byType(TaskProgressBar));
        expect(progress.value, 0.25);
        expect(progress.height, 3);
        expect(progress.shimmer, isTrue);
        expect(find.text('128 KB/s'), findsOneWidget);
        expect(find.text('archive.zip'), findsOneWidget);
        expect(find.text('1 MB / 4 MB'), findsOneWidget);
        expect(find.text('30 seconds remaining'), findsOneWidget);
        expect(find.text('Connection Status'), findsOneWidget);
        expect(find.byType(TaskStatisticsTab), findsNothing);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
      });
    }
  }

  testWidgets('pause, resume and completion follow live task state', (tester) async {
    final tasks = await mount(tester);
    expect(find.text('Cancel Download'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('download-progress-pause')));
    await tester.pump();
    expect(tasks.actions, ['pause:download']);
    expect(find.byKey(const ValueKey('download-progress-resume')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('download-progress-resume')));
    await tester.pump();
    expect(tasks.actions, ['pause:download', 'resume:download']);
    tasks.emit(TaskStatus.completed);
    await tester.pump();
    expect(find.text('100.0%'), findsNothing);
    expect(tester.widget<TaskProgressBar>(find.byType(TaskProgressBar)).value, 1.0);
    expect(find.byKey(const ValueKey('download-progress-pause')), findsNothing);
    expect(find.byKey(const ValueKey('download-progress-resume')), findsNothing);
    expect(find.text('Cancel Download'), findsNothing);
    expect(tasks.actions, hasLength(2));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('failed pause is reported and restores controls', (tester) async {
    final tasks = await mount(tester);
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

  for (final width in [320.0, AppDesignTokens.downloadProgressWindowSize.width]) {
    testWidgets(
      'connection list scrolls inside an anchored native window at width $width',
      (tester) async {
        const windowChannel = MethodChannel('window_manager');
        const screenChannel = MethodChannel('dev.leanflutter.plugins/screen_retriever');
        final messenger = tester.binding.defaultBinaryMessenger;
        final compactHeight = width == 320 ? 320.0 : AppDesignTokens.downloadProgressWindowSize.height;
        final compact = Rect.fromLTWH(820, 210, width, compactHeight);
        final displayHeight = width == 320 ? 900.0 : 570.0;
        final expandedHeight = width == 320 ? AppDesignTokens.downloadProgressExpandedHeight : 400.0;
        final expanded = compact.topLeft & Size(width, expandedHeight);
        var bounds = compact;
        final resizes = <Rect>[];
        messenger.setMockMethodCallHandler(windowChannel, (call) async {
          switch (call.method) {
            case 'isMaximized':
            case 'isFullScreen':
              return false;
            case 'getBounds':
              return {'x': bounds.left, 'y': bounds.top, 'width': bounds.width, 'height': bounds.height};
            case 'setBounds':
              final args = call.arguments as Map;
              expect(args.containsKey('x'), isFalse);
              expect(args.containsKey('y'), isFalse);
              bounds = bounds.topLeft & Size(args['width'], args['height']);
              resizes.add(bounds);
              tester.view.physicalSize = bounds.size;
              return null;
          }
          throw MissingPluginException(call.method);
        });
        messenger.setMockMethodCallHandler(screenChannel, (call) async {
          expect(call.method, 'getAllDisplays');
          return {
            'displays': [
              {
                'id': 'primary',
                'size': {'width': 800.0, 'height': 900.0},
                'visiblePosition': {'dx': 0.0, 'dy': 0.0},
              },
              {
                'id': 'secondary',
                'size': {'width': 800.0, 'height': displayHeight + 40},
                'visiblePosition': {'dx': 800.0, 'dy': 40.0},
                'visibleSize': {'width': 800.0, 'height': displayHeight},
              },
            ],
          };
        });
        addTearDown(() {
          messenger.setMockMethodCallHandler(windowChannel, null);
          messenger.setMockMethodCallHandler(screenChannel, null);
        });

        final stats = HttpTaskStats(
          connections: List.filled(
            12,
            const HttpConnectionStats(downloaded: 2, total: 5, completed: false, failed: false, retryTimes: 0),
          ),
        );
        await mount(tester, size: compact.size, stats: stats);
        final details = find.byKey(const ValueKey('download-progress-details'));
        final pause = find.byKey(const ValueKey('download-progress-pause'));
        final cancel = find.byKey(const ValueKey('download-progress-cancel'));
        expect(tester.getCenter(pause).dx, greaterThan(tester.getCenter(details).dx));
        expect(tester.getCenter(cancel).dx, greaterThan(tester.getCenter(details).dx));
        if (width == AppDesignTokens.downloadProgressWindowSize.width) {
          expect(tester.getCenter(details).dy, tester.getCenter(pause).dy);
          expect(tester.getCenter(details).dy, tester.getCenter(cancel).dy);
        }
        final controlsTop = tester.getTopLeft(details).dy;
        await tester.tap(details);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 250));
        expect(resizes, [expanded]);
        expect(find.byType(TaskStatisticsTab), findsOneWidget);
        expect(find.text('Download Link'), findsNothing);
        expect(find.text('Storage Path'), findsNothing);
        final connections = find.byType(DownloadProgressConnections);
        final scrollbar = find.descendant(of: connections, matching: find.byType(Scrollbar));
        expect(tester.widget<Scrollbar>(scrollbar).thumbVisibility, isFalse);
        final scrollable = find.descendant(of: connections, matching: find.byType(Scrollable));
        await tester.drag(scrollable, const Offset(0, -80));
        await tester.pump(const Duration(milliseconds: 300));
        expect(tester.state<ScrollableState>(scrollable).position.pixels, greaterThan(0));
        expect(tester.getTopLeft(details).dy, controlsTop);
        expect(pause.hitTestable(), findsOneWidget);
        expect(cancel.hitTestable(), findsOneWidget);

        await tester.tap(details);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 250));
        expect(resizes, [expanded, compact]);
        expect(find.byType(TaskStatisticsTab), findsNothing);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
      },
      variant: TargetPlatformVariant({TargetPlatform.macOS}),
    );
  }

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
        final tasks = await mount(tester, size: Size(width, 440));
        final openFile = find.byKey(const ValueKey('download-progress-open-file'));
        final openFolder = find.byKey(const ValueKey('download-progress-open-folder'));
        expect(openFile, findsNothing);
        expect(openFolder, findsNothing);
        tasks.emit(TaskStatus.completed);
        await tester.pump();
        expect(openFile.hitTestable(), findsOneWidget);
        expect(openFolder.hitTestable(), findsOneWidget);
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

  testWidgets(
    'completed directory action closes only after opening succeeds',
    (tester) async {
      var closed = false;
      final tasks = await mount(tester, onClose: () => closed = true);
      tasks.emit(TaskStatus.completed);
      await tester.pump();
      final opened = Completer<bool>();
      const channel = MethodChannel('com.flutter/open-dir');
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.method, 'openNativeDir');
        expect((call.arguments as Map)['highlightedFileName'], 'archive.zip');
        return opened.future;
      });
      addTearDown(() => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, null));
      await IOOverrides.runZoned(() async {
        await tester.tap(find.byKey(const ValueKey('download-progress-open-folder')));
        await tester.pump();
        expect(closed, isFalse);
        opened.complete(true);
        await tester.pump();
        expect(closed, isTrue);
        expect(tasks.actions, isEmpty);
      }, fseGetType: (path, followLinks) async => FileSystemEntityType.file);
      await tester.pumpWidget(const SizedBox());
    },
    variant: TargetPlatformVariant({TargetPlatform.windows, TargetPlatform.macOS, TargetPlatform.linux}),
  );

  for (final reveal in [false, true]) {
    testWidgets(
      'completed ${reveal ? 'folder' : 'file'} action uses task path and reports missing files',
      (tester) async {
        var closed = false;
        final tasks = await mount(tester, onClose: () => closed = true);
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
            expect(closed, isFalse);
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
