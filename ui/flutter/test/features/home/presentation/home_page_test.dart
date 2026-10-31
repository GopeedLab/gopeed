import 'dart:io';

import 'package:flutter/gestures.dart' show kSecondaryMouseButton;
import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gopeed/api/model/task.dart' as api;
import 'package:gopeed/features/home/presentation/pages/home_page.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/pages/task_details_page.dart';
import 'package:gopeed/features/tasks/presentation/pages/task_files_page.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_card/task_card.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_drawer.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:gopeed/core/common/api_server_state.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_file_manager.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

api.Task member(String id, {String name = 'member.txt', bool bt = false, String status = 'running'}) =>
    api.Task.fromJson({
      'id': id,
      'name': name,
      'parentId': 'group',
      'group': {'id': 'group', 'name': 'bundle', 'path': '/downloads/bundle', 'createdAt': '2026-01-01T00:00:00Z'},
      'protocol': bt ? 'bt' : 'http',
      'status': status,
      'uploading': false,
      'createdAt': '2026-01-01T00:00:00Z',
      'updatedAt': '2026-01-01T00:00:00Z',
      'meta': {
        'req': {'url': bt ? 'magnet:?xt=urn:btih:test' : 'https://example.com/$id'},
        'opts': {'path': '/downloads/bundle', 'name': name, 'size': 100},
        'res': {
          'name': bt ? name : '',
          'size': 100,
          'files': [
            {'name': bt ? 'bt-first.txt' : name, 'size': 50},
            if (bt) {'name': 'bt-second.txt', 'size': 50},
          ],
        },
      },
      'progress': {'used': 0, 'speed': 0, 'downloaded': 25, 'uploaded': 0, 'uploadSpeed': 0},
    });

class GroupTasks extends TasksController {
  String name = 'member.txt';
  String childStatus = 'running';
  final paused = <String>[];
  final resumed = <String>[];
  final deleted = <String>[];

  TasksState snapshot() => TasksState(
    tasks: taskRecordsFromApi([
      member('child', name: name, status: childStatus),
      member('torrent', name: 'torrent bundle', bt: true, status: 'pause'),
    ]),
  );

  @override
  Future<TasksState> build() async => snapshot();

  void rename() {
    name = 'renamed.txt';
    state = AsyncData(snapshot());
  }

  void completeMember() {
    childStatus = 'done';
    state = AsyncData(snapshot());
  }

  @override
  Future<void> pause(String id) async => paused.add(id);
  @override
  Future<void> pauseAll(List<String>? ids) async => paused.addAll(ids!);
  @override
  Future<void> resume(String id) async => resumed.add(id);
  @override
  Future<void> resumeAll(List<String>? ids) async => resumed.addAll(ids!);
  @override
  Future<void> deleteSelected(List<String>? ids, {bool force = false}) async {
    deleted.addAll(ids!);
    state = AsyncData(
      TasksState(
        tasks: taskRecordsFromApi([
          if (!deleted.contains('child') && !deleted.contains('group')) member('child', name: name),
          if (!deleted.contains('torrent') && !deleted.contains('group'))
            member('torrent', name: 'torrent bundle', bt: true, status: 'pause'),
        ]),
      ),
    );
  }
}

class GroupRuntime extends AppRuntimeController {
  @override
  Future<AppRuntimeState> build() async => AppRuntimeState(
    startConfig: StartConfig(),
    apiServerState: const ApiServerState(
      enabled: false,
      mcpEnabled: false,
      running: false,
      network: '',
      address: '',
      runningPort: 0,
      pendingApply: false,
      lastError: '',
    ),
    downloaderConfig: DownloaderConfig(),
  );
  @override
  Future<void> reloadConfig() async {}
}

Future<void> settle(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 500));
}

Finder card(String id) => find.byWidgetPredicate((widget) => widget is TaskCard && widget.task.id == id);

Future<({GroupTasks controller, GoRouter router})> showHome(WidgetTester tester, double width) async {
  tester.view.physicalSize = Size(width, 900);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  final router = GoRouter(
    routes: [
      GoRoute(path: '/', builder: (_, _) => const HomePage()),
      GoRoute(
        path: '/tasks/:id',
        builder: (_, state) =>
            TaskDetailsPage(taskId: state.pathParameters['id']!, initialTask: state.extra as TaskRecord?),
      ),
      GoRoute(
        path: '/tasks/:id/files',
        builder: (_, state) =>
            TaskFilesPage(taskId: state.pathParameters['id']!, initialTask: state.extra as TaskRecord?),
      ),
    ],
  );
  addTearDown(router.dispose);
  late GroupTasks controller;
  final registry = CapabilityRegistry(createAppCapabilityCodecs())
    ..bind(GopeedMethods.putConfig, (_) => const RpcUnit());
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tasksControllerProvider.overrideWith(() => controller = GroupTasks()),
        appRuntimeControllerProvider.overrideWith(GroupRuntime.new),
        appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry))),
      ],
      child: shad.ShadcnApp.router(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        routerConfig: router,
      ),
    ),
  );
  await settle(tester);
  return (controller: controller, router: router);
}

void main() {
  for (final width in [390.0, 700.0, 1100.0]) {
    testWidgets('group expands actual member tasks and keeps a multi-file BT as one card at $width', (tester) async {
      final app = await showHome(tester, width);
      expect(card('group'), findsOneWidget);
      expect(card('child'), findsNothing);
      await tester.tap(find.text('bundle'));
      await settle(tester);
      expect(card('child'), findsOneWidget);
      expect(card('torrent'), findsOneWidget);
      expect(find.text('bt-first.txt'), findsNothing);
      expect(find.text('bt-second.txt'), findsNothing);
      expect(find.byType(TaskFileManagerView), findsNothing);
      expect(find.byType(TaskDetailsView), findsNothing);
      app.controller.rename();
      await settle(tester);
      expect(tester.widget<TaskCard>(card('child')).task.name, 'renamed.txt');
      await tester.tap(find.text('bundle'));
      await settle(tester);
      expect(card('child'), findsNothing);
      expect(card('torrent'), findsNothing);
      expect(tester.takeException(), isNull);
    });

    testWidgets('member click opens its own details and polling refreshes the selected member at $width', (
      tester,
    ) async {
      final app = await showHome(tester, width);
      await tester.tap(find.text('bundle'));
      await settle(tester);
      await tester.tap(find.text('member.txt'));
      await settle(tester);
      expect(tester.widget<TaskDetailsView>(find.byType(TaskDetailsView).last).task.id, 'child');
      app.controller.rename();
      await settle(tester);
      expect(tester.widget<TaskDetailsView>(find.byType(TaskDetailsView).last).task.name, 'renamed.txt');
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('member context menu and card actions target member IDs while group actions keep the group ID', (
    tester,
  ) async {
    final app = await showHome(tester, 1100);
    await tester.tap(find.text('bundle'));
    await settle(tester);
    await tester.tapAt(tester.getCenter(card('child')), buttons: kSecondaryMouseButton);
    await settle(tester);
    await tester.tap(find.byKey(const ValueKey('task-context-pause')));
    await settle(tester);
    expect(app.controller.paused, ['child']);
    tester.widget<TaskCard>(card('torrent')).onResume!();
    await settle(tester);
    expect(app.controller.resumed, ['torrent']);
    tester.widget<TaskCard>(card('group')).onPause!();
    await settle(tester);
    expect(app.controller.paused, ['child', 'group']);
    expect(tester.takeException(), isNull);
  });

  for (final width in [390.0, 1100.0]) {
    testWidgets('group resume button targets the group at $width', (tester) async {
      final app = await showHome(tester, width);
      await tester.tap(find.descendant(of: card('group'), matching: find.byIcon(Icons.play_arrow)));
      await settle(tester);
      expect(app.controller.resumed, ['group']);
      expect(card('child'), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  for (final open in [true, false]) {
    testWidgets('member card and menu ${open ? 'open' : 'reveal'} use the member path', (tester) async {
      final app = await showHome(tester, 1100);
      app.controller.completeMember();
      await settle(tester);
      await tester.tap(find.text('bundle'));
      await settle(tester);
      final checkedPaths = <String>[];
      final child = tester.widget<TaskCard>(card('child'));
      await IOOverrides.runZoned(
        () async {
          (open ? child.onOpen : child.onReveal)!();
          await settle(tester);
          expect(checkedPaths.first, '/downloads/bundle/member.txt');
          checkedPaths.clear();
          await tester.tapAt(tester.getCenter(card('child')), buttons: kSecondaryMouseButton);
          await settle(tester);
          await tester.tap(find.byKey(ValueKey(open ? 'task-context-open-file' : 'task-context-open-directory')));
          await settle(tester);
          expect(checkedPaths.first, '/downloads/bundle/member.txt');
        },
        fseGetType: (path, followLinks) async {
          checkedPaths.add(path);
          return FileSystemEntityType.notFound;
        },
      );
      // Allow the failure-path toast to close without launching native tools.
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 6));
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('member menu details uses the member instead of the group', (tester) async {
    await showHome(tester, 1100);
    await tester.tap(find.text('bundle'));
    await settle(tester);
    await tester.tapAt(tester.getCenter(card('torrent')), buttons: kSecondaryMouseButton);
    await settle(tester);
    await tester.tap(find.byKey(const ValueKey('task-context-details')));
    await settle(tester);
    expect(tester.widget<TaskDetailsView>(find.byType(TaskDetailsView).last).task.id, 'torrent');
    expect(tester.widget<TaskDetailsView>(find.byType(TaskDetailsView).last).task.files, hasLength(2));
    expect(tester.takeException(), isNull);
  });

  testWidgets('selecting a member through its context menu enables batch actions for that member', (tester) async {
    final app = await showHome(tester, 1100);
    await tester.tap(find.text('bundle'));
    await settle(tester);
    await tester.tapAt(tester.getCenter(card('child')), buttons: kSecondaryMouseButton);
    await settle(tester);
    await tester.tap(find.byKey(const ValueKey('task-context-select')));
    await settle(tester);
    expect(tester.widget<TaskCard>(card('child')).selectedInBatch, isTrue);
    final pause = find.byIcon(Icons.pause_outlined);
    await tester.tap(pause);
    await settle(tester);
    expect(app.controller.paused, ['child']);
    expect(tester.takeException(), isNull);
  });

  testWidgets('deleting a member leaves its sibling group intact', (tester) async {
    final app = await showHome(tester, 1100);
    await tester.tap(find.text('bundle'));
    await settle(tester);
    await tester.tapAt(tester.getCenter(card('child')), buttons: kSecondaryMouseButton);
    await settle(tester);
    await tester.tap(find.byKey(const ValueKey('task-context-delete')));
    await settle(tester);
    expect(find.byType(shad.DestructiveButton), findsOneWidget);
    await tester.tap(find.byType(shad.DestructiveButton));
    await settle(tester);
    expect(app.controller.deleted, ['child']);
    expect(card('group'), findsOneWidget);
    expect(card('child'), findsNothing);
    expect(card('torrent'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('deleting a selected group and its members counts and submits the group once', (tester) async {
    final app = await showHome(tester, 1100);
    await tester.tap(find.text('bundle'));
    await settle(tester);
    await tester.tapAt(tester.getCenter(card('group')), buttons: kSecondaryMouseButton);
    await settle(tester);
    await tester.tap(find.byKey(const ValueKey('task-context-select-all')));
    await settle(tester);
    await tester.tapAt(tester.getCenter(card('child')), buttons: kSecondaryMouseButton);
    await settle(tester);
    await tester.tap(find.byKey(const ValueKey('task-context-delete')));
    await settle(tester);
    expect(find.text('Delete task?'), findsOneWidget);
    await tester.tap(find.byType(shad.DestructiveButton));
    await settle(tester);
    expect(app.controller.deleted, ['group']);
    expect(card('group'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('member context file browser polls its member record', (tester) async {
    final app = await showHome(tester, 1100);
    await tester.tap(find.text('bundle'));
    await settle(tester);
    await tester.tapAt(tester.getCenter(card('child')), buttons: kSecondaryMouseButton);
    await settle(tester);
    await tester.tap(find.byKey(const ValueKey('task-context-browse-files')));
    await settle(tester);
    expect(tester.widget<TaskFileManagerView>(find.byType(TaskFileManagerView)).task.name, 'member.txt');
    app.controller.rename();
    await settle(tester);
    expect(tester.widget<TaskFileManagerView>(find.byType(TaskFileManagerView)).task.name, 'renamed.txt');
    expect(tester.takeException(), isNull);
  });

  testWidgets('group clicks in batch mode select without collapsing members', (tester) async {
    await showHome(tester, 1100);
    await tester.tap(find.text('bundle'));
    await settle(tester);
    await tester.tapAt(tester.getCenter(card('child')), buttons: kSecondaryMouseButton);
    await settle(tester);
    await tester.tap(find.byKey(const ValueKey('task-context-select')));
    await settle(tester);
    await tester.tap(find.text('bundle'));
    await settle(tester);
    expect(tester.widget<TaskCard>(card('group')).selectedInBatch, isTrue);
    expect(tester.widget<TaskCard>(card('group')).expanded, isTrue);
    expect(card('child'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('member detail route resolves the member without an initial snapshot', (tester) async {
    final app = await showHome(tester, 390);
    app.router.go('/tasks/child');
    await settle(tester);
    expect(tester.widget<TaskDetailsView>(find.byType(TaskDetailsView).last).task.id, 'child');
    app.controller.rename();
    await settle(tester);
    expect(tester.widget<TaskDetailsView>(find.byType(TaskDetailsView).last).task.name, 'renamed.txt');
    expect(tester.takeException(), isNull);
  });

  testWidgets('member file page resolves current data from its group without a route snapshot', (tester) async {
    final app = await showHome(tester, 390);
    app.router.go('/tasks/child/files');
    await settle(tester);
    expect(tester.widget<TaskFileManagerView>(find.byType(TaskFileManagerView)).task.name, 'member.txt');
    app.controller.rename();
    await settle(tester);
    expect(tester.widget<TaskFileManagerView>(find.byType(TaskFileManagerView)).task.name, 'renamed.txt');
    expect(tester.takeException(), isNull);
  });
}
