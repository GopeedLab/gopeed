import 'dart:async';
import 'dart:io';
import 'dart:math' as math;

import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter/foundation.dart' show defaultTargetPlatform;
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/task_stats.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/storage_capability.dart';
import 'package:gopeed/core/window/app_window_bootstrap.dart';
import 'package:gopeed/core/window/app_window_payload.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/tasks/application/task_stats_provider.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/pages/download_progress_window_page.dart';
import 'package:gopeed/features/tasks/presentation/widgets/download_progress_connections.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_progress_bar.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_info_row.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_statistics/task_statistics_tab.dart';
import 'package:gopeed/l10n/l10n.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/theme/app_design_tokens.dart';

void main() {
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    final font = FontLoader('packages/shadcn_flutter/GeistSans');
    for (final weight in ['Regular', 'Medium', 'Bold']) {
      font.addFont(rootBundle.load('packages/shadcn_flutter/lib/fonts/Geist-$weight.otf'));
    }
    await font.load();
  });

  testWidgets(
    'the native progress window disables resizing and skips unsupported shadows',
    (tester) async {
      const window = MethodChannel('window_manager');
      const screen = MethodChannel('dev.leanflutter.plugins/screen_retriever');
      const display = {
        'id': 'primary',
        'size': {'width': 1920.0, 'height': 1080.0},
        'visiblePosition': {'dx': 0.0, 'dy': 0.0},
      };
      final messenger = tester.binding.defaultBinaryMessenger;
      final calls = <MethodCall>[];
      messenger.setMockMethodCallHandler(window, (call) async {
        calls.add(call);
        return switch (call.method) {
          'isFullScreen' || 'isMaximized' || 'isMinimized' => false,
          'getBounds' => {'x': 0.0, 'y': 0.0, 'width': 500.0, 'height': 224.0},
          _ => null,
        };
      });
      messenger.setMockMethodCallHandler(
        screen,
        (call) async => switch (call.method) {
          'getAllDisplays' => {
            'displays': [display],
          },
          'getCursorScreenPoint' => {'dx': 0.0, 'dy': 0.0},
          'getPrimaryDisplay' => display,
          _ => null,
        },
      );
      addTearDown(() {
        messenger.setMockMethodCallHandler(window, null);
        messenger.setMockMethodCallHandler(screen, null);
      });

      await AppWindowBootstrap.setupSubWindow(AppWindowType.downloadProgress, localeConfig: 'en');
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 180));
      Iterable<dynamic> arguments(String method) =>
          calls.where((call) => call.method == method).map((call) => call.arguments);
      expect(arguments('setResizable').single, {'isResizable': false});
      expect(arguments('setMaximizable').single, {'isMaximizable': false});
      expect(
        arguments('setHasShadow'),
        defaultTargetPlatform == TargetPlatform.linux
            ? isEmpty
            : [
                {'hasShadow': true},
              ],
      );
      expect(calls.map((call) => call.method), containsAllInOrder(['setMaximizable', 'show', 'focus']));
    },
    variant: TargetPlatformVariant({TargetPlatform.linux, TargetPlatform.windows, TargetPlatform.macOS}),
  );

  Future<_Tasks> mount(
    WidgetTester tester, {
    Size size = AppDesignTokens.downloadProgressWindowSize,
    VoidCallback? onClose,
    bool dark = false,
    TaskStats? stats,
    Future<TasksState>? firstSnapshot,
    CapabilityRegistry? registry,
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final container = ProviderContainer(
      overrides: [
        if (registry != null)
          appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry))),
        tasksControllerProvider.overrideWith(() => _Tasks(firstSnapshot)),
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

  testWidgets('renders the field labels before the first snapshot and fills them in place', (tester) async {
    final first = Completer<TasksState>();
    final tasks = await mount(tester, firstSnapshot: first.future);
    final rows = find.byType(TaskInfoRow);
    final emptyValues = find.descendant(of: rows, matching: find.text(''));
    final labels = ['Status', 'Size', 'Speed', 'Remaining'];
    for (final label in labels) {
      expect(find.text(label), findsOneWidget);
    }
    expect(emptyValues, findsNWidgets(4));
    final initialPositions = [for (final label in labels) tester.getTopLeft(find.text(label))];
    first.complete(TasksState(tasks: [_task(TaskStatus.downloading, speed: null, remainingSeconds: null)]));
    await tester.idle();
    await tester.pump();
    expect(find.text('1 MB / 4 MB'), findsOneWidget);
    expect(emptyValues, findsNWidgets(2));
    tasks.emit(TaskStatus.downloading);
    await tester.pump();
    expect(find.text('128 KB/s'), findsOneWidget);
    expect(find.text('30 seconds remaining'), findsOneWidget);
    expect([for (final label in labels) tester.getTopLeft(find.text(label))], initialPositions);
    tasks.emit(TaskStatus.completed);
    await tester.pump();
    expect(find.text('Remaining'), findsNothing);
    expect(find.text('Download duration'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
  });

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
        expect(find.text('archive.zip'), findsNothing);
        expect(find.text('Download Link'), findsNothing);
        expect(find.text('https://example.com/archive.zip'), findsNothing);
        final speed = tester.element(find.text('128 KB/s'));
        expect(MediaQuery.textScalerOf(speed).scale(12), AppDesignTokens.desktopMenuFontSize);
        expect(find.text('1 MB / 4 MB'), findsOneWidget);
        final remaining = find.text('30 seconds remaining');
        expect(remaining, findsOneWidget);
        final barTop = tester.getTopLeft(find.byType(TaskProgressBar)).dy;
        await tester.drag(
          find.ancestor(of: remaining, matching: find.byType(SingleChildScrollView)),
          const Offset(0, -160),
        );
        await tester.pump(const Duration(milliseconds: 300));
        expect(tester.getTopLeft(find.byType(TaskProgressBar)).dy, barTop);
        expect(barTop - tester.getBottomLeft(remaining).dy, greaterThanOrEqualTo(AppDesignTokens.space12));
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

  testWidgets('unknown size, download error and polling error are visible', (tester) async {
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
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('only removing the displayed task closes the window', (tester) async {
    var closes = 0;
    final other = _task(TaskStatus.downloading, id: 'other');
    final tasks = await mount(
      tester,
      onClose: () => closes++,
      firstSnapshot: Future.value(TasksState(tasks: [other, _task(TaskStatus.downloading)])),
    );
    tasks.emit(TaskStatus.paused);
    await tester.pump();
    tasks.emit(TaskStatus.completed);
    await tester.pump();
    expect(closes, 0);
    tasks.replace(TasksState(tasks: [other]));
    await tester.pump();
    tasks.remove();
    await tester.pump();
    expect(closes, 1);
    expect(tasks.actions, isEmpty);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('a task removed during startup closes after the first successful refresh', (tester) async {
    var closes = 0;
    final first = Completer<TasksState>();
    final tasks = await mount(tester, firstSnapshot: first.future, onClose: () => closes++);
    expect(closes, 0);
    first.completeError(StateError('offline'));
    await tester.pump();
    tasks.replace(TasksState(tasks: const [], lastError: StateError('offline')));
    await tester.pump();
    expect(closes, 0);
    tasks.remove();
    await tester.pump();
    expect(closes, 1);
    await tester.pumpWidget(const SizedBox());
  });

  for (final width in [320.0, 500.0]) {
    testWidgets(
      'progress and connections grow past the screen edge in place at width $width',
      (tester) async {
        const windowChannel = MethodChannel('window_manager');
        const screenChannel = MethodChannel('dev.leanflutter.plugins/screen_retriever');
        final messenger = tester.binding.defaultBinaryMessenger;
        const screenBottom = 900.0;
        final initialHeight = AppDesignTokens.downloadProgressWindowSize.height;
        var bounds = Rect.fromLTWH(80, screenBottom - initialHeight, width, initialHeight);
        var displayQueries = 0;
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
          displayQueries++;
          return {
            'displays': [
              {
                'id': 'primary',
                'size': {'width': 800.0, 'height': screenBottom},
                'visiblePosition': {'dx': 0.0, 'dy': 0.0},
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
        final tasks = await mount(tester, size: bounds.size, stats: stats);
        await tester.pump();
        final compact = bounds;
        resizes.clear();
        final details = find.byKey(const ValueKey('download-progress-details'));
        final pause = find.byKey(const ValueKey('download-progress-pause'));
        final cancel = find.byKey(const ValueKey('download-progress-cancel'));
        final bar = find.byType(TaskProgressBar);
        if (width == 500) {
          expect(compact.height, 224);
          tasks.replace(TasksState(tasks: [_task(TaskStatus.downloading, uploading: true)]));
          await tester.pump();
          await tester.pump();
          final fields = find.ancestor(of: find.text('Upload speed'), matching: find.byType(SingleChildScrollView));
          final fieldViewport = tester.getRect(fields);
          expect(tester.getRect(bar).top, greaterThan(fieldViewport.bottom));
          expect(tester.getRect(bar).bottom, lessThan(tester.getRect(details).top));
          expect(bounds.topLeft, compact.topLeft);
          expect(bounds.bottom, greaterThan(screenBottom));
          expect(tester.getRect(find.text('30 seconds remaining')).bottom, lessThanOrEqualTo(fieldViewport.bottom));
          tasks.emit(TaskStatus.downloading);
          await tester.pump();
          await tester.pump();
          expect(bounds, compact);
          resizes.clear();
        }
        expect(tester.getCenter(pause).dx, greaterThan(tester.getCenter(details).dx));
        expect(tester.getCenter(cancel).dx, greaterThan(tester.getCenter(details).dx));
        if (width == AppDesignTokens.downloadProgressWindowSize.width) {
          expect(tester.getCenter(details).dy, tester.getCenter(pause).dy);
          expect(tester.getCenter(details).dy, tester.getCenter(cancel).dy);
          final controlsBottom = math.max(tester.getBottomLeft(details).dy, tester.getBottomLeft(pause).dy);
          expect(compact.height - controlsBottom, closeTo(AppDesignTokens.space8, 2));
        }
        final controlsTop = tester.getTopLeft(details).dy;
        await tester.tap(details);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 250));
        final expanded = bounds;
        expect(expanded.topLeft, compact.topLeft);
        expect(expanded.width, compact.width);
        final expectedHeight = math.min(
          AppDesignTokens.downloadProgressExpandedHeight,
          compact.height + AppDesignTokens.downloadProgressConnectionsHeight + AppDesignTokens.space8,
        );
        expect(expanded.height, expectedHeight);
        expect(expanded.bottom, greaterThan(screenBottom));
        expect(find.byType(TaskStatisticsTab), findsOneWidget);
        expect(find.text('Download Link'), findsNothing);
        expect(find.text('Storage Path'), findsNothing);
        final connections = find.byType(DownloadProgressConnections);
        expect(tester.getBottomLeft(connections).dy, expanded.height - AppDesignTokens.space8);
        expect(
          tester.getSize(connections).height,
          lessThanOrEqualTo(AppDesignTokens.downloadProgressConnectionsHeight),
        );
        if (width == 500) {
          expect(tester.getSize(connections).height, AppDesignTokens.downloadProgressConnectionsHeight);
        }
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
        expect(displayQueries, 0);
        expect(find.byType(TaskStatisticsTab), findsNothing);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
      },
      variant: TargetPlatformVariant({TargetPlatform.windows, TargetPlatform.macOS, TargetPlatform.linux}),
    );
  }

  for (final deleteFiles in [false, true]) {
    testWidgets('cancel restores the shared switch and remembers file deletion=$deleteFiles', (tester) async {
      var closes = 0;
      var keepFiles = deleteFiles;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(StorageMethods.getLastDeleteTaskKeep, (_) => keepFiles)
        ..bind(StorageMethods.saveLastDeleteTaskKeep, (value) {
          keepFiles = value;
          return const RpcUnit();
        });
      final tasks = await mount(tester, onClose: () => closes++, registry: registry);
      final cancel = find.byKey(const ValueKey('download-progress-cancel'));
      final deleteSwitch = find.byKey(const ValueKey('task-delete-files-switch'));
      await tester.tap(cancel);
      await tester.pump(const Duration(milliseconds: 300));
      expect(tester.widget<Switch>(deleteSwitch).value, !deleteFiles);
      await tester.tap(deleteSwitch);
      await tester.pump();
      await tester.tap(find.byType(SecondaryButton));
      await tester.pump(const Duration(milliseconds: 300));
      expect(keepFiles, deleteFiles);
      expect(tasks.actions, isEmpty);
      expect(closes, 0);

      await tester.tap(cancel);
      await tester.pump(const Duration(milliseconds: 300));
      expect(tester.widget<Switch>(deleteSwitch).value, !deleteFiles);
      await tester.tap(deleteSwitch);
      await tester.pump();
      await tester.tap(find.byType(DestructiveButton));
      await tester.pump(const Duration(milliseconds: 300));
      expect(keepFiles, !deleteFiles);
      expect(tasks.actions, ['delete:download:$deleteFiles']);
      expect(closes, 1);
      await tester.pumpWidget(const SizedBox());
    });
  }

  for (final width in [320.0, 520.0]) {
    testWidgets(
      'completed file actions stay visible at compact width $width',
      (tester) async {
        final tasks = await mount(tester, size: Size(width, AppDesignTokens.downloadProgressWindowSize.height));
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
  _Tasks(this.firstSnapshot);

  final Future<TasksState>? firstSnapshot;
  final actions = <String>[];
  Completer<void>? pauseRequest;
  @override
  Future<void> deleteSelected(List<String>? ids, {bool force = false}) async {
    actions.add('delete:${ids!.single}:$force');
    remove();
  }

  @override
  Future<TasksState> build() async => await firstSnapshot ?? TasksState(tasks: [_task(TaskStatus.downloading)]);
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
  void replace(TasksState snapshot) => state = AsyncData(snapshot);
  void remove() => replace(const TasksState(tasks: []));
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
  String id = 'download',
  bool unknownSize = false,
  String? speed = '128 KB/s',
  int? remainingSeconds = 30,
  String? error,
  bool isFolder = false,
  bool isGroup = false,
  bool uploading = false,
}) => TaskRecord(
  id: id,
  name: 'archive.zip',
  status: status,
  isFolder: isFolder,
  isGroup: isGroup,
  downloaded: '1 MB',
  total: unknownSize ? null : '4 MB',
  progress: unknownSize ? null : 0.25,
  speed: speed,
  speedBytes: 131072,
  downloadedBytes: 1048576,
  downloadDuration: const Duration(seconds: 16),
  remainingSeconds: remainingSeconds,
  error: error,
  url: 'https://example.com/archive.zip',
  storagePath: '/downloads/archive.zip',
  files: const [],
  uploading: uploading,
  uploadSpeed: uploading ? '64 KB/s' : null,
);
