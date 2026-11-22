import 'dart:async';

import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/create_task_batch.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/app/app.dart';
import 'package:gopeed/app/application/app_deep_link_controller.dart';
import 'package:gopeed/app/application/app_notification_controller.dart';
import 'package:gopeed/app/application/app_platform_controller.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/app/router/app_router.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/capabilities/storage_capability.dart';
import 'package:gopeed/core/common/api_server_state.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:gopeed/core/window/window_capability_transport.dart';
import 'package:gopeed/features/home/presentation/pages/home_page.dart';
import 'package:gopeed/features/tasks/application/task_list_navigation.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/tasks/presentation/pages/create_task_window_page.dart';
import 'package:gopeed/shared/widgets/app_text_field.dart';

void main() {
  const windowChannel = MethodChannel('mixin.one/desktop_multi_window');
  const channels = MethodChannel('mixin.one/desktop_multi_window/channels');
  const windowManagerChannel = MethodChannel('window_manager');
  final binding = TestWidgetsFlutterBinding.ensureInitialized();
  final messenger = TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  setUp(() {
    binding.platformDispatcher.localesTestValue = const [Locale('en')];
    messenger.setMockMethodCallHandler(windowChannel, (_) async => throw PlatformException(code: 'unavailable'));
    messenger.setMockMethodCallHandler(channels, (_) async => null);
    messenger.setMockMethodCallHandler(
      windowManagerChannel,
      (call) async => call.method == 'isMinimized' ? false : null,
    );
  });
  tearDown(() async {
    await AppWindowCapabilityHost.instance.stop();
    messenger.setMockMethodCallHandler(windowChannel, null);
    messenger.setMockMethodCallHandler(channels, null);
    messenger.setMockMethodCallHandler(windowManagerChannel, null);
    binding.platformDispatcher.clearLocalesTestValue();
  });

  Future<ProviderContainer> mount(WidgetTester tester, {CapabilityRegistry? registry}) async {
    tester.view.physicalSize = const Size(1024, 768);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    registry ??= CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'));
    final container = ProviderContainer(
      overrides: [
        appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry))),
        appRuntimeControllerProvider.overrideWith(_Runtime.new),
        appPlatformControllerProvider.overrideWith(_Platform.new),
        appDeepLinkControllerProvider.overrideWith(_DeepLinks.new),
        appNotificationControllerProvider.overrideWith(_Notifications.new),
        tasksControllerProvider.overrideWith(_Tasks.new),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(UncontrolledProviderScope(container: container, child: const GopeedApp()));
    await tester.pumpAndSettle();
    return container;
  }

  Future<void> unmount(WidgetTester tester) async {
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  }

  testWidgets(
    'modal cancel and history restore preserve the mounted task list',
    (tester) async {
      await mount(tester);
      final router = GoRouter.of(AppRouter.rootNavigatorKey.currentContext!);
      final home = tester.state(find.byType(HomePage));
      final search = find.descendant(
        of: find.byKey(const ValueKey('tasks-search-field-container')),
        matching: find.byType(AppTextField),
      );
      await tester.enterText(search, 'keep this filter');
      await tester.pumpAndSettle();
      final before = router.routeInformationProvider.value;
      await tester.tap(find.byKey(const ValueKey('tasks-create-button-container')));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('app-window-modal')), findsOneWidget);
      expect(find.byType(HomePage, skipOffstage: false), findsOneWidget);
      final opened = router.routeInformationProvider.value;
      final modalBounds = tester.getRect(find.byKey(const ValueKey('app-window-modal')));
      expect(modalBounds.size, const Size(820, 620));
      final padding = tester.widget<Padding>(find.byKey(const ValueKey('create-task-safe-content')));
      expect(padding.padding, EdgeInsets.zero);

      await router.routeInformationProvider.didPushRouteInformation(before);
      await tester.pumpAndSettle();
      expect(find.byType(CreateTaskWindowPage), findsNothing);
      expect(tester.state(find.byType(HomePage)), same(home));
      expect(tester.widget<AppTextField>(search).controller!.text, 'keep this filter');
      await router.routeInformationProvider.didPushRouteInformation(opened);
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('app-window-modal')), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey('create-task-cancel-button')));
      await tester.pumpAndSettle();
      expect(tester.state(find.byType(HomePage)), same(home));
      expect(tester.widget<AppTextField>(search).controller!.text, 'keep this filter');
      await unmount(tester);
    },
    variant: TargetPlatformVariant({TargetPlatform.macOS}),
  );

  testWidgets('modal fits phone, tablet and desktop viewports', (tester) async {
    await mount(tester);
    final router = GoRouter.of(AppRouter.rootNavigatorKey.currentContext!);
    router.go('/create');
    await tester.pumpAndSettle();
    for (final size in [const Size(375, 667), const Size(768, 700), const Size(1280, 900)]) {
      tester.view.physicalSize = size;
      await tester.pumpAndSettle();
      final bounds = tester.getRect(find.byKey(const ValueKey('app-window-modal')));
      expect(bounds.left, greaterThan(0));
      expect(bounds.top, greaterThanOrEqualTo(0));
      expect(bounds.right, lessThan(size.width));
      expect(bounds.bottom, lessThanOrEqualTo(size.height));
      expect(find.byKey(const ValueKey('create-task-confirm-button')).hitTestable(), findsOneWidget);
      expect(tester.takeException(), isNull);
    }
    tester.view.physicalSize = const Size(375, 667);
    tester.view.viewInsets = const FakeViewPadding(bottom: 260);
    addTearDown(tester.view.resetViewInsets);
    await tester.pumpAndSettle();
    final confirmBounds = tester.getRect(find.byKey(const ValueKey('create-task-confirm-button')));
    expect(confirmBounds.bottom, lessThanOrEqualTo(667 - 260));
    tester.view.resetViewInsets();
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('create-task-cancel-button')));
    await tester.pumpAndSettle();
    expect(find.byType(HomePage), findsOneWidget);
    await unmount(tester);
  }, variant: TargetPlatformVariant({TargetPlatform.macOS}));

  testWidgets(
    'failed submission keeps input and retry submits once then selects downloads',
    (tester) async {
      var attempts = 0;
      Completer<CreateTaskBatchResult>? pending;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(
          GopeedMethods.getConfig,
          (_) => DownloaderConfig(downloadDir: '/downloads')..extra.defaultDirectDownload = true,
        )
        ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit())
        ..bind(GopeedMethods.createTaskBatch, (_) {
          attempts++;
          if (attempts == 1) throw StateError('offline');
          pending = Completer<CreateTaskBatchResult>();
          return pending!.future;
        });
      final container = await mount(tester, registry: registry);
      await AppRouter.openCreateTask(
        createTask: CreateTask(req: Request(url: 'https://example.com/file.zip')),
      );
      await tester.pumpAndSettle();
      final url = find.byKey(const ValueKey('create-task-url-input'));
      expect(tester.widget<AppTextField>(url).controller!.text, 'https://example.com/file.zip');
      final confirm = find.byKey(const ValueKey('create-task-confirm-button'));
      await tester.tap(confirm);
      await tester.pumpAndSettle();
      expect(attempts, 1);
      await tester.pump(const Duration(seconds: 6));
      await tester.pumpAndSettle();
      expect(tester.widget<AppTextField>(url).controller!.text, 'https://example.com/file.zip');
      expect(find.byKey(const ValueKey('app-window-modal')), findsOneWidget);
      await tester.tap(confirm);
      await tester.pump();
      await tester.tap(confirm);
      await tester.pump();
      expect(attempts, 2);
      pending!.complete(CreateTaskBatchResult(taskIds: ['new-task']));
      await tester.pumpAndSettle();
      expect(find.byType(CreateTaskWindowPage), findsNothing);
      expect(container.read(taskListNavigationProvider), 1);
      expect(find.byType(HomePage), findsOneWidget);
      await unmount(tester);
    },
    variant: TargetPlatformVariant({TargetPlatform.macOS}),
  );

  testWidgets(
    'file preview can cancel back to the modal and submit the resolved selection',
    (tester) async {
      CreateTaskBatch? submitted;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
        ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit())
        ..bind(
          GopeedMethods.resolve,
          (_) => ResolveResult(
            id: 'resolved-torrent',
            res: Resource(
              name: 'bundle',
              files: [
                FileInfo(name: 'first.zip', size: 10),
                FileInfo(name: 'second.zip', size: 20),
              ],
            ),
          ),
        )
        ..bind(GopeedMethods.createTaskBatch, (batch) {
          submitted = batch;
          return CreateTaskBatchResult(taskIds: ['torrent-task']);
        });
      await mount(tester, registry: registry);
      await AppRouter.openCreateTask(
        createTask: CreateTask(req: Request(url: 'magnet:?xt=urn:btih:test')),
      );
      await tester.pumpAndSettle();
      final confirm = find.byKey(const ValueKey('create-task-confirm-button'));
      await tester.tap(confirm);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(find.text('first.zip'), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('app-window-modal')), findsOneWidget);
      expect(submitted, isNull);
      await tester.tap(confirm);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
      await tester.pumpAndSettle();
      expect(submitted!.reqs!.single.rid, 'resolved-torrent');
      expect(submitted!.reqs!.single.opts!.selectFiles, [0, 1]);
      expect(find.byType(CreateTaskWindowPage), findsNothing);
      expect(find.byType(HomePage), findsOneWidget);
      await unmount(tester);
    },
    variant: TargetPlatformVariant({TargetPlatform.macOS}),
  );

  testWidgets('cancel returns to the section underneath the modal', (tester) async {
    await mount(tester);
    final router = GoRouter.of(AppRouter.rootNavigatorKey.currentContext!);
    router.go('/settings');
    await tester.pumpAndSettle();
    await AppRouter.openCreateTask();
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('app-window-modal')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('create-task-cancel-button')));
    await tester.pumpAndSettle();
    expect(router.state.uri.path, '/settings');
    await unmount(tester);
  }, variant: TargetPlatformVariant({TargetPlatform.macOS}));

  testWidgets(
    'native mobile opens a page and updates repeated share requests',
    (tester) async {
      await mount(tester);
      await AppRouter.openCreateTask(
        createTask: CreateTask(req: Request(url: 'https://example.com/first.zip')),
      );
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('app-window-modal')), findsNothing);
      expect(find.byType(CreateTaskWindowPage), findsOneWidget);
      await AppRouter.openCreateTask(
        createTask: CreateTask(req: Request(url: 'https://example.com/second.zip')),
      );
      await tester.pumpAndSettle();
      final url = find.byKey(const ValueKey('create-task-url-input'));
      expect(tester.widget<AppTextField>(url).controller!.text, 'https://example.com/second.zip');
      await tester.tap(find.byKey(const ValueKey('create-task-cancel-button')));
      await tester.pumpAndSettle();
      expect(find.byType(HomePage), findsOneWidget);
      await unmount(tester);
    },
    skip: kIsWeb,
    variant: TargetPlatformVariant({TargetPlatform.android, TargetPlatform.iOS}),
  );

  testWidgets('logo returns home with mouse and keyboard activation', (tester) async {
    await mount(tester);
    final router = GoRouter.of(AppRouter.rootNavigatorKey.currentContext!);
    final logo = find.byKey(const ValueKey('primary-rail-home-button'));
    for (final keyboard in [false, true]) {
      router.go('/settings');
      await tester.pumpAndSettle();
      if (keyboard) {
        Focus.of(tester.element(find.byKey(const ValueKey('primary-rail-app-mark')))).requestFocus();
        await tester.pump();
        await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      } else {
        await tester.tap(logo);
      }
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/');
      expect(find.byType(HomePage), findsOneWidget);
    }
    await unmount(tester);
  }, variant: TargetPlatformVariant({TargetPlatform.macOS}));
}

class _Runtime extends AppRuntimeController {
  @override
  Future<AppRuntimeState> build() async => AppRuntimeState(
    startConfig: StartConfig()
      ..network = 'tcp'
      ..address = '127.0.0.1:9999'
      ..apiEnable = true
      ..apiToken = ''
      ..storage = 'bolt'
      ..storageDir = ''
      ..refreshInterval = 0,
    apiServerState: ApiServerState.fromJson({}),
    downloaderConfig: DownloaderConfig()..extra.locale = 'en',
  );
}

class _Platform extends AppPlatformController {
  @override
  Future<AppPlatformState> build() async => const AppPlatformState(started: true);
}

class _DeepLinks extends AppDeepLinkController {
  @override
  Future<AppDeepLinkState> build() async => const AppDeepLinkState(started: true);
}

class _Notifications extends AppNotificationController {
  @override
  Future<AppNotificationState> build() async => const AppNotificationState(started: true);
}

class _Tasks extends TasksController {
  @override
  Future<TasksState> build() async => const TasksState(tasks: []);
}
