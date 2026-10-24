import 'dart:convert';

import 'package:flutter/widgets.dart';
import 'package:flutter/services.dart';
import 'package:desktop_multi_window/desktop_multi_window.dart';
import 'package:flutter/widgets.dart' as flutter show Row;
import 'package:flutter/foundation.dart' show TargetPlatform, debugDefaultTargetPlatformOverride;
import 'package:flutter/material.dart' show Icons;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/create_task_batch.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/api/model/resolve_task.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/app_navigation_capability.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/capabilities/storage_capability.dart';
import 'package:gopeed/features/tasks/presentation/pages/create_task_window_page.dart';
import 'package:gopeed/features/tasks/presentation/widgets/resolve_file_tree.dart';
import 'package:gopeed/features/tasks/application/pending_create_task.dart';
import 'package:gopeed/shared/services/download_directory_picker.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_design_tokens.dart';
import 'package:gopeed/shared/theme/app_palette.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_loading_button.dart';
import 'package:gopeed/shared/widgets/app_tooltip.dart';
import 'package:gopeed/l10n/app_localizations.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('extension file estimates survive batch submission and window serialization', (tester) async {
    tester.view.physicalSize = const Size(900, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    CreateTaskBatch? submitted;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(
        GopeedMethods.resolve,
        (_) => ResolveResult(
          id: '',
          res: Resource(
            files: [
              FileInfo(
                name: 'under.mp4',
                size: 3,
                req: Request(url: 'https://example.com/video'),
              ),
              FileInfo(
                name: 'over.mp4',
                size: 1048576,
                req: Request(url: 'https://example.com/audio'),
              ),
            ],
          ),
        ),
      )
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        submitted = CreateTaskBatch.fromJson(jsonDecode(jsonEncode(batch)) as Map<String, dynamic>);
        return ['first', 'second'];
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    final router = GoRouter(
      initialLocation: '/create',
      routes: [
        GoRoute(path: '/', builder: (_, _) => const SizedBox()),
        GoRoute(path: '/create', builder: (_, _) => const CreateTaskWindowPage()),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
        child: shad.ShadcnApp.router(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          routerConfig: router,
          builder: (_, child) => AppComponentThemes(child: child!),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/watch');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pumpAndSettle();
    expect(submitted?.reqs?.map((item) => item.opts?.size), [3, 1048576]);
    expect(submitted?.reqs?.map((item) => item.req?.url), ['https://example.com/video', 'https://example.com/audio']);
    expect(tester.takeException(), isNull);
  });

  for (final navigationFails in [false, true]) {
    testWidgets('child creation selects downloads before closing (navigation fails: $navigationFails)', (tester) async {
      tester.view.physicalSize = const Size(1024, 720);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final events = <String>[];
      const channel = MethodChannel('window_manager');
      final messenger = TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
      messenger.setMockMethodCallHandler(channel, (call) async {
        events.add(call.method);
        return null;
      });
      addTearDown(() => messenger.setMockMethodCallHandler(channel, null));
      final config = DownloaderConfig(downloadDir: '/downloads')..extra.defaultDirectDownload = true;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(GopeedMethods.getConfig, (_) => config)
        ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit())
        ..bind(GopeedMethods.createTask, (_) {
          events.add('created');
          return 'task-id';
        })
        ..bind(NavigationMethods.showDownloadingTasks, (_) {
          events.add('downloads');
          if (navigationFails) throw StateError('Navigation unavailable');
          return const RpcUnit();
        });
      await tester.pumpWidget(
        ProviderScope(
          overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
          child: shad.ShadcnApp(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            home: AppComponentThemes(
              child: CreateTaskWindowPage(
                windowController: WindowController.fromWindowId('child'),
                initialTask: CreateTask(req: Request(url: 'https://example.com/file.apk')),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Confirm'));
      // The mocked native close deliberately leaves the widget mounted.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(events, ['created', 'downloads', 'close']);
      await tester.pumpWidget(const SizedBox());
    });
  }

  testWidgets('repeated external tasks update the open create route and replace request metadata', (tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    try {
      tester.view.physicalSize = const Size(700, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      CreateTask? submitted;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
        ..bind(GopeedMethods.createTask, (task) {
          submitted = task;
          return 'created-id';
        })
        ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
      final container = ProviderContainer(
        overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
      );
      addTearDown(container.dispose);
      final pending = container.read(pendingCreateTaskProvider.notifier);
      pending.set(
        CreateTask(
          req: Request(
            url: 'https://example.com/first.zip',
            rawUrl: 'https://example.com/first',
            labels: {'source': 'first'},
            extra: ReqExtraHttp(method: 'POST', body: 'old-body', header: {'Cookie': 'old-cookie'}).toJson(),
            proxy: RequestProxy(mode: RequestProxyMode.custom, host: 'localhost:8080'),
            skipVerifyCert: true,
          ),
        ),
      );
      final router = GoRouter(
        initialLocation: '/create',
        routes: [
          GoRoute(path: '/', builder: (_, _) => const SizedBox()),
          GoRoute(path: '/create', builder: (_, _) => const CreateTaskWindowPage()),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: shad.ShadcnApp.router(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            routerConfig: router,
            builder: (_, child) => AppComponentThemes(child: child!),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(_fieldText(tester, 'create-task-url-input'), 'https://example.com/first.zip');
      expect(container.read(pendingCreateTaskProvider), isNull);
      final pageState = tester.state(find.byType(CreateTaskWindowPage));

      for (final url in [
        'https://example.com/second.zip',
        'https://example.com/third.zip',
        'https://example.com/third.zip',
      ]) {
        pending.set(CreateTask(req: Request(url: url)));
        router.go('/create');
        await tester.pumpAndSettle();
        expect(tester.state(find.byType(CreateTaskWindowPage)), same(pageState));
        expect(_fieldText(tester, 'create-task-url-input'), url);
        expect(container.read(pendingCreateTaskProvider), isNull);
      }
      await tester.tap(find.byKey(const ValueKey('create-task-direct-download-toggle')));
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();
      expect(submitted?.req?.url, 'https://example.com/third.zip');
      expect(submitted?.req?.rawUrl, isNull);
      expect(submitted?.req?.labels, isNull);
      expect(submitted?.req?.extra, isNull);
      expect(submitted?.req?.proxy?.mode, RequestProxyMode.follow);
      expect(submitted?.req?.skipVerifyCert, isFalse);
      expect(tester.takeException(), isNull);
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });

  for (final omitOptions in [false, true]) {
    testWidgets('an incoming task without a directory shows the configured directory (omit options: $omitOptions)', (
      tester,
    ) async {
      CreateTask? submitted;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(
          GopeedMethods.getConfig,
          (_) => DownloaderConfig(downloadDir: '/downloads')..extra.defaultDirectDownload = true,
        )
        ..bind(GopeedMethods.createTask, (task) {
          submitted = task;
          return 'created-id';
        })
        ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
      final container = ProviderContainer(
        overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
      );
      addTearDown(container.dispose);
      final pending = container.read(pendingCreateTaskProvider.notifier);
      final router = GoRouter(
        initialLocation: '/create',
        routes: [
          GoRoute(path: '/', builder: (_, _) => const SizedBox()),
          GoRoute(path: '/create', builder: (_, _) => const CreateTaskWindowPage()),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: shad.ShadcnApp.router(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            routerConfig: router,
            builder: (_, child) => AppComponentThemes(child: child!),
          ),
        ),
      );
      await tester.pumpAndSettle();

      pending.set(
        CreateTask(
          req: Request(url: 'https://example.com/first.zip'),
          opts: Options(path: 'D:/Other', skipAutoCategorize: true),
        ),
      );
      router.go('/create');
      await tester.pumpAndSettle();
      expect(_fieldText(tester, 'create-task-directory-input'), 'D:/Other');

      // The second task names no directory, so the field may not keep the one the
      // user chose for the first task.
      pending.set(
        CreateTask(
          req: Request(url: 'https://example.com/second.zip'),
          opts: omitOptions ? null : Options(),
        ),
      );
      router.go('/create');
      await tester.pumpAndSettle();
      expect(_fieldText(tester, 'create-task-directory-input'), '/downloads');

      await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));
      expect(submitted?.opts?.path, '/downloads');
      expect(submitted?.opts?.skipAutoCategorize, isFalse);
      await _settleToastTimers(tester);
    });
  }

  testWidgets('mobile create task page stays within the system safe areas', (WidgetTester tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    tester.view.physicalSize = const Size(700, 760);
    tester.view.devicePixelRatio = 1;
    tester.view.padding = const FakeViewPadding(top: 32, bottom: 24);
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetPadding);

    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'));
    final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

    try {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
          child: shad.ShadcnApp(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            home: AppComponentThemes(child: const CreateTaskWindowPage()),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final contentPadding = tester.widget<Padding>(find.byKey(const ValueKey('create-task-safe-content')));
      expect((contentPadding.padding as EdgeInsets).top, 32);
      expect(
        tester.getBottomRight(find.byKey(const ValueKey('create-task-confirm-button'))).dy,
        lessThanOrEqualTo(tester.view.physicalSize.height / tester.view.devicePixelRatio - 24),
      );
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });

  testWidgets('browser extension request parameters populate the form and survive submission', (
    WidgetTester tester,
  ) async {
    tester.view.physicalSize = const Size(1024, 720);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    DownloadDirectoryPicker.debugPlatformOverride = TargetPlatform.android;
    final androidLocations = <String, String>{'application': 'D:/Downloads', 'downloads': 'G:/Manually selected'};
    DownloadDirectoryPicker.debugAndroidLocationsLoader = () async => androidLocations;
    DownloadDirectoryPicker.debugDownloadsPreparer = (path) async => path;
    addTearDown(() {
      DownloadDirectoryPicker.debugPlatformOverride = null;
      DownloadDirectoryPicker.debugAndroidLocationsLoader = null;
      DownloadDirectoryPicker.debugDownloadsPreparer = null;
    });

    CreateTask? submitted;
    final config = DownloaderConfig(downloadDir: 'C:/Downloads')
      ..extra.defaultDirectDownload = true
      ..categories = [
        DownloadCategory(name: 'Archives', path: 'E:/Archives/%year%'),
        DownloadCategory(name: 'Archives', path: 'F:/Archives'),
        DownloadCategory(name: 'Long video category', path: 'G:/Videos'),
        DownloadCategory(name: 'Long document category', path: 'H:/Documents'),
      ]
      ..protocolConfig.http.connections = 8;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => config)
      ..bind(GopeedMethods.createTask, (task) {
        submitted = task;
        return 'created-task';
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

    final initialTask = CreateTask(
      req: Request(
        rawUrl: 'https://page.example/download',
        url: 'https://cdn.example/archive.zip',
        extra: ReqExtraHttp(
          method: 'POST',
          header: const {
            'User-Agent': 'Browser UA',
            'Cookie': 'session=abc',
            'Referer': 'https://page.example/',
            'Authorization': 'Bearer token',
          },
          body: 'request-body',
        ).toJson(),
        labels: const {'source': 'browser-extension'},
        proxy: RequestProxy(
          mode: RequestProxyMode.custom,
          scheme: 'socks5',
          host: '127.0.0.1:7890',
          usr: 'proxy-user',
          pwd: 'proxy-password',
        ),
        skipVerifyCert: true,
      ),
      opts: Options(
        size: 1234,
        name: 'archive.zip',
        path: 'D:/Downloads',
        extra: OptsExtraHttp(
          connections: 12,
          autoTorrent: true,
          deleteTorrentAfterDownload: true,
          autoExtract: true,
          archivePassword: 'archive-password',
          deleteAfterExtract: true,
        ).toJson(),
      ),
    );
    final router = GoRouter(
      initialLocation: '/create',
      routes: [
        GoRoute(path: '/', builder: (_, _) => const SizedBox()),
        GoRoute(
          path: '/create',
          builder: (_, _) => CreateTaskWindowPage(initialTask: initialTask),
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
        child: shad.ShadcnApp.router(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          builder: (context, child) => AppComponentThemes(child: child ?? const SizedBox.shrink()),
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    final connectionsInput = find.byKey(const ValueKey('create-task-connections-input'));
    expect(connectionsInput, findsOneWidget);
    expect(
      tester.getSize(connectionsInput).width,
      tester.getSize(find.byKey(const ValueKey('create-task-rename-input'))).width,
    );
    final connectionsField = tester.widget<shad.TextField>(connectionsInput);
    expect(connectionsField.controller!.text, '12');
    expect(connectionsField.features.single, isA<shad.InputSpinnerFeature>());

    final advancedSection = find.byKey(const ValueKey('create-task-advanced-section'));
    expect(tester.widget<AnimatedCrossFade>(advancedSection).crossFadeState, CrossFadeState.showSecond);
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    expect(tester.widget<AnimatedCrossFade>(advancedSection).crossFadeState, CrossFadeState.showFirst);

    expect(_fieldText(tester, 'create-task-http-header-name-0'), 'User-Agent');
    expect(_fieldText(tester, 'create-task-http-header-value-0'), 'Browser UA');
    expect(_fieldText(tester, 'create-task-http-header-name-3'), 'Authorization');
    expect(_fieldText(tester, 'create-task-http-header-value-3'), 'Bearer token');
    expect(_fieldText(tester, 'create-task-proxy-server'), '127.0.0.1');
    expect(_fieldText(tester, 'create-task-proxy-port'), '7890');
    expect(_fieldText(tester, 'create-task-proxy-username'), 'proxy-user');
    expect(_fieldText(tester, 'create-task-proxy-password'), 'proxy-password');
    expect(_fieldText(tester, 'create-task-archive-password'), 'archive-password');
    expect(find.byKey(const ValueKey('create-task-skip-verify-cert')), findsOneWidget);
    expect(find.byKey(const ValueKey('create-task-auto-torrent')), findsOneWidget);
    expect(find.byKey(const ValueKey('create-task-delete-torrent')), findsOneWidget);
    expect(find.byKey(const ValueKey('create-task-auto-extract')), findsOneWidget);
    expect(find.byKey(const ValueKey('create-task-delete-after-extract')), findsOneWidget);

    expect(find.text('Archives'), findsNWidgets(2));
    expect(find.text('Quick folders'), findsNothing);
    expect(find.byKey(const ValueKey('create-task-category-0')), findsOneWidget);
    expect(find.byKey(const ValueKey('create-task-category-1')), findsOneWidget);
    expect(find.byKey(const ValueKey('create-task-category-2')), findsOneWidget);
    expect(find.byKey(const ValueKey('create-task-category-3')), findsOneWidget);
    final shortcutRow = find.byKey(const ValueKey('create-task-directory-shortcuts'));
    expect(tester.getSize(shortcutRow).height, 28);
    final rememberDirectory = find.byKey(const ValueKey('create-task-remember-download-directory-checkbox'));
    final directDownloadCheckbox = find.byKey(const ValueKey('create-task-direct-download-checkbox'));
    final rememberDirectoryRow = find.byKey(const ValueKey('create-task-remember-download-directory'));
    final rememberDirectoryLabel = find.byKey(const ValueKey('create-task-remember-download-directory-label'));
    final directDownloadLabel = find.descendant(
      of: find.byKey(const ValueKey('create-task-direct-download-toggle')),
      matching: find.text('Skip resolving and create tasks immediately'),
    );
    final directoryComponent = find.byKey(const ValueKey('create-task-directory-component'));
    final directoryOptionsRow = find.byKey(const ValueKey('create-task-directory-options-row'));
    final directoryOptionsLayout = find.byKey(const ValueKey('create-task-directory-options-layout'));
    final directoryPicker = find.byKey(const ValueKey('create-task-directory-picker'));
    expect(tester.widget<shad.Checkbox>(rememberDirectory).state, shad.CheckboxState.unchecked);
    expect(find.text('Save as default download directory'), findsOneWidget);
    final rememberStyle = tester.widget<Text>(rememberDirectoryLabel).style!;
    final directStyle = tester.widget<Text>(directDownloadLabel).style!;
    expect(rememberStyle.color, directStyle.color);
    expect(rememberStyle.fontSize, directStyle.fontSize);
    expect(rememberStyle.fontWeight, directStyle.fontWeight);
    expect(tester.widget<shad.Checkbox>(rememberDirectory).size, isNull);
    expect(tester.getSize(rememberDirectory), const Size.square(AppDesignTokens.checkboxSize));
    expect(tester.getCenter(rememberDirectory).dx, closeTo(tester.getCenter(directDownloadCheckbox).dx, 0.5));
    final rememberDirectoryText = tester.widget<Text>(rememberDirectoryLabel);
    final directDownloadText = tester.widget<Text>(directDownloadLabel);
    expect(rememberDirectoryText.style, directDownloadText.style);
    expect(rememberDirectoryText.maxLines, directDownloadText.maxLines);
    expect(rememberDirectoryText.overflow, directDownloadText.overflow);
    expect(tester.widget<shad.IconButton>(directoryPicker).variance, same(shad.ButtonVariance.outline));
    expect(tester.widget<flutter.Row>(directoryOptionsLayout).mainAxisAlignment, MainAxisAlignment.spaceBetween);
    expect(tester.getCenter(rememberDirectoryRow).dy, closeTo(tester.getCenter(shortcutRow).dy, 0.5));
    expect(tester.getTopRight(directoryOptionsRow).dx, closeTo(tester.getTopRight(directoryComponent).dx, 0.5));
    expect(
      tester.getTopRight(find.byKey(const ValueKey('create-task-category-3'))).dx,
      closeTo(tester.getTopRight(directoryComponent).dx, 0.5),
    );

    tester.view.physicalSize = const Size(520, 720);
    await tester.pumpAndSettle();
    expect(tester.getTopLeft(shortcutRow).dy, greaterThan(tester.getBottomLeft(rememberDirectoryRow).dy));
    expect(tester.getSize(shortcutRow).width, closeTo(tester.getSize(directoryComponent).width, 0.5));
    expect(find.descendant(of: shortcutRow, matching: find.byType(Scrollable)), findsNothing);
    for (var index = 0; index < 4; index++) {
      final shortcut = tester.getRect(find.byKey(ValueKey('create-task-category-$index')));
      expect(shortcut.left, greaterThanOrEqualTo(tester.getRect(shortcutRow).left));
      expect(shortcut.right, lessThanOrEqualTo(tester.getRect(shortcutRow).right));
    }

    tester.view.physicalSize = const Size(719, 720);
    await tester.pumpAndSettle();
    expect(
      tester.widget<shad.Column>(find.byKey(const ValueKey('create-task-directory-options-layout'))),
      isA<shad.Column>(),
    );

    tester.view.physicalSize = const Size(720, 720);
    await tester.pumpAndSettle();
    expect(
      tester.widget<flutter.Row>(find.byKey(const ValueKey('create-task-directory-options-layout'))).mainAxisAlignment,
      MainAxisAlignment.spaceBetween,
    );

    tester.view.physicalSize = const Size(1024, 720);
    await tester.pumpAndSettle();

    await tester.tap(directoryPicker);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('android-app-directory-option')));
    await tester.pumpAndSettle();
    // Picking the directory already shown is still an explicit choice, so the
    // default toggle is offered and pre-checked for a non-default directory.
    expect(tester.widget<shad.Checkbox>(rememberDirectory).state, shad.CheckboxState.checked);

    await tester.tap(directoryPicker);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('android-downloads-directory-option')));
    await tester.pumpAndSettle();
    expect(tester.widget<shad.Checkbox>(rememberDirectory).state, shad.CheckboxState.checked);

    expect(tester.takeException(), isNull);
    await tester.tap(find.byKey(const ValueKey('create-task-category-3')));
    await tester.pump();
    expect(tester.widget<shad.Checkbox>(rememberDirectory).state, shad.CheckboxState.unchecked);
    expect(
      find.descendant(
        of: find.byKey(const ValueKey('create-task-category-3')),
        matching: find.byIcon(Icons.folder_outlined),
      ),
      findsOneWidget,
    );

    androidLocations['downloads'] = 'C:/Downloads';
    await tester.tap(directoryPicker);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('android-downloads-directory-option')));
    await tester.pumpAndSettle();
    expect(tester.widget<shad.Checkbox>(rememberDirectory).state, shad.CheckboxState.unchecked);

    androidLocations['downloads'] = 'G:/Manually selected';
    await tester.tap(directoryPicker);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('android-downloads-directory-option')));
    await tester.pumpAndSettle();
    expect(tester.widget<shad.Checkbox>(rememberDirectory).state, shad.CheckboxState.checked);

    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();

    final request = submitted?.req;
    expect(request, isNotNull);
    expect(request?.rawUrl, initialTask.req?.rawUrl);
    expect(request?.labels, initialTask.req?.labels);
    expect(request?.proxy?.mode, RequestProxyMode.custom);
    expect(request?.proxy?.scheme, 'socks5');
    expect(request?.proxy?.host, '127.0.0.1:7890');
    expect(request?.proxy?.usr, 'proxy-user');
    expect(request?.proxy?.pwd, 'proxy-password');
    expect(request?.skipVerifyCert, isTrue);
    expect(ReqExtraHttp.fromJson(request?.extra! as Map<String, dynamic>).toJson(), initialTask.req?.extra);
    expect(submitted?.opts?.path, 'G:/Manually selected');
    expect(submitted?.opts?.size, 1234);
    final options = OptsExtraHttp.fromJson(submitted?.opts?.extra! as Map<String, dynamic>);
    expect(options.connections, 12);
    expect(options.autoTorrent, isTrue);
    expect(options.deleteTorrentAfterDownload, isTrue);
    expect(options.autoExtract, isTrue);
    expect(options.archivePassword, 'archive-password');
    expect(options.deleteAfterExtract, isTrue);
    expect(submitted?.opts?.asDefaultPath, isTrue);
  });

  testWidgets('resolved task keeps rename empty and uses equal-width single-line actions', (WidgetTester tester) async {
    tester.view.physicalSize = const Size(700, 700);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(
        GopeedMethods.resolve,
        (_) => ResolveResult(
          id: 'resolved-id',
          res: Resource(
            name: 'server-generated-name',
            files: [FileInfo(name: 'video.mp4', size: 1024)],
          ),
        ),
      )
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(child: const CreateTaskWindowPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/video.mp4');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(_fieldText(tester, 'create-task-rename-input'), isEmpty);
    final createButton = tester.widget<AppLoadingButton>(find.byKey(const ValueKey('resolve-create-button')));
    expect(createButton.child, isA<SizedBox>());
    expect((createButton.child as SizedBox).width, 68);
    expect(
      find.descendant(of: find.byKey(const ValueKey('resolve-create-button')), matching: find.text('Create')),
      findsOneWidget,
    );
    expect(
      tester.getSize(find.byKey(const ValueKey('resolve-cancel-button'))),
      tester.getSize(find.byKey(const ValueKey('resolve-create-button'))),
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('create history supports filtering, deleting one entry, and clearing all entries', (
    WidgetTester tester,
  ) async {
    tester.view.physicalSize = const Size(1024, 720);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final histories = <String>[
      'https://example.com/alpha.zip',
      'https://example.com/beta.iso',
      'magnet:?xt=urn:btih:gamma',
    ];
    String? removedHistory;
    var cleared = false;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(StorageMethods.getCreateHistory, (_) => List<String>.unmodifiable(histories))
      ..bind(StorageMethods.removeCreateHistory, (value) {
        removedHistory = value;
        histories.remove(value);
        return const RpcUnit();
      })
      ..bind(StorageMethods.clearCreateHistory, (_) {
        cleared = true;
        histories.clear();
        return const RpcUnit();
      });
    final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(child: const CreateTaskWindowPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.history));
    await tester.pumpAndSettle();
    expect(find.text('History Links'), findsOneWidget);
    expect(
      tester.getSize(find.byKey(const ValueKey('create-history-title-bar'))).width,
      tester.getSize(find.byKey(const ValueKey('create-history-content'))).width,
    );
    final alphaItem = find.byKey(const ValueKey('create-history-item-https://example.com/alpha.zip'));
    final betaItem = find.byKey(const ValueKey('create-history-item-https://example.com/beta.iso'));
    final gammaItem = find.byKey(const ValueKey('create-history-item-magnet:?xt=urn:btih:gamma'));
    expect(alphaItem, findsOneWidget);
    expect(betaItem, findsOneWidget);
    expect(gammaItem, findsOneWidget);
    final clearAction = find.byKey(const ValueKey('create-history-clear'));
    final closeAction = find.byKey(const ValueKey('create-history-close'));
    expect(
      tester.widget<AppTooltip>(find.ancestor(of: clearAction, matching: find.byType(AppTooltip))).message,
      'Clear History',
    );
    expect(
      tester.widget<AppTooltip>(find.ancestor(of: closeAction, matching: find.byType(AppTooltip))).message,
      'Close',
    );
    final historyText = tester.widget<Text>(find.descendant(of: alphaItem, matching: find.byType(Text)));
    expect(historyText.data, contains('\u200B'));
    expect(historyText.semanticsLabel, 'https://example.com/alpha.zip');

    final filter = find.byKey(const ValueKey('create-history-filter'));
    await tester.enterText(filter, 'beta');
    await tester.pump();
    expect(alphaItem, findsNothing);
    expect(betaItem, findsOneWidget);
    expect(gammaItem, findsNothing);

    await tester.tap(find.byKey(const ValueKey('create-history-delete-https://example.com/beta.iso')));
    await tester.pumpAndSettle();
    expect(removedHistory, 'https://example.com/beta.iso');
    expect(find.text('No History Found'), findsOneWidget);

    await tester.enterText(filter, '');
    await tester.pump();
    expect(alphaItem, findsOneWidget);
    expect(gammaItem, findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('create-history-clear')));
    await tester.pumpAndSettle();
    expect(cleared, isTrue);
    expect(find.text('No history'), findsOneWidget);
    expect(find.byKey(const ValueKey('create-history-clear')), findsOneWidget);
    expect(find.byKey(const ValueKey('create-history-close')), findsOneWidget);
  });

  testWidgets('the untouched default directory leaves the decision to the backend', (tester) async {
    CreateTask? submitted;
    await _pumpCreateTaskWithCategories(
      tester,
      downloadDir: 'C:/Downloads',
      url: 'https://example.com/setup.exe',
      categories: [
        DownloadCategory(name: 'Program', path: 'C:/Downloads/Program', extensions: const ['exe']),
      ],
      onCreate: (task) => submitted = task,
    );

    expect(_fieldText(tester, 'create-task-directory-input'), 'C:/Downloads');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(submitted?.opts?.path, 'C:/Downloads');
    expect(submitted?.opts?.skipAutoCategorize, isFalse);
    await _settleToastTimers(tester);
  });

  testWidgets('a manual directory choice skips auto categorize', (tester) async {
    CreateTask? submitted;
    await _pumpCreateTaskWithCategories(
      tester,
      downloadDir: 'C:/Downloads',
      url: 'https://example.com/setup.exe',
      categories: [
        DownloadCategory(name: 'Program', path: 'C:/Downloads/Program', extensions: const ['exe']),
      ],
      onCreate: (task) => submitted = task,
    );

    await tester.enterText(find.byKey(const ValueKey('create-task-directory-input')), 'D:/Other');
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(submitted?.opts?.path, 'D:/Other');
    expect(submitted?.opts?.skipAutoCategorize, isTrue);
    await _settleToastTimers(tester);
  });

  testWidgets('typing the default directory back still skips auto categorize', (tester) async {
    CreateTask? submitted;
    await _pumpCreateTaskWithCategories(
      tester,
      downloadDir: 'C:/Downloads',
      url: 'https://example.com/setup.exe',
      categories: [
        DownloadCategory(name: 'Program', path: 'C:/Downloads/Program', extensions: const ['exe']),
      ],
      onCreate: (task) => submitted = task,
    );

    final field = find.byKey(const ValueKey('create-task-directory-input'));
    await tester.enterText(field, 'D:/Other');
    await tester.pumpAndSettle();
    await tester.enterText(field, 'C:/Downloads');
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pumpAndSettle();

    // The value matches the default directory, but the user still named it.
    expect(submitted?.opts?.path, 'C:/Downloads');
    expect(submitted?.opts?.skipAutoCategorize, isTrue);
    await _settleToastTimers(tester);
  });

  testWidgets('picking the directory that is already shown still skips auto categorize', (tester) async {
    CreateTask? submitted;
    await _pumpCreateTaskWithCategories(
      tester,
      downloadDir: 'C:/Downloads',
      url: 'https://example.com/setup.exe',
      categories: [
        DownloadCategory(name: 'Program', path: 'C:/Downloads/Program', extensions: const ['exe']),
      ],
      onCreate: (task) => submitted = task,
    );
    DownloadDirectoryPicker.debugPlatformOverride = TargetPlatform.android;
    DownloadDirectoryPicker.debugAndroidLocationsLoader = () async => {
      'application': 'C:/Downloads',
      'downloads': 'C:/Public',
    };
    addTearDown(() => DownloadDirectoryPicker.debugAndroidLocationsLoader = null);

    await tester.tap(find.byKey(const ValueKey('create-task-directory-picker')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('android-app-directory-option')));
    await tester.pumpAndSettle();

    // The picked directory equals the one already shown, but picking is an
    // explicit choice, so the task skips the classification.
    final checkbox = find.byKey(const ValueKey('create-task-remember-download-directory-checkbox'));
    expect(tester.widget<shad.Checkbox>(checkbox).onChanged, isNotNull);
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(submitted?.opts?.path, 'C:/Downloads');
    expect(submitted?.opts?.skipAutoCategorize, isTrue);
    await _settleToastTimers(tester);
  });

  testWidgets('checksum state is reset for each incoming task and not inherited (T1)', (tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    try {
      tester.view.physicalSize = const Size(700, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      CreateTask? submitted;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
        ..bind(GopeedMethods.createTask, (task) {
          submitted = task;
          return 'task-id';
        })
        ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());

      final container = ProviderContainer(
        overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
      );
      addTearDown(container.dispose);

      final pending = container.read(pendingCreateTaskProvider.notifier);
      // Task A: with checksum
      pending.set(
        CreateTask(
          req: Request(url: 'https://example.com/taskA.zip'),
          opts: Options(
            extra: OptsExtraHttp(
              checksum: ChecksumOption(
                algorithm: 'sha256',
                expected: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
              ),
            ).toJson(),
          ),
        ),
      );

      final router = GoRouter(
        initialLocation: '/create',
        routes: [
          GoRoute(path: '/', builder: (_, _) => const SizedBox()),
          GoRoute(path: '/create', builder: (_, _) => const CreateTaskWindowPage()),
        ],
      );
      addTearDown(router.dispose);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: shad.ShadcnApp.router(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            routerConfig: router,
            builder: (_, child) => AppComponentThemes(child: child!),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Expand advanced
      await tester.tap(find.text('Advanced'));
      await tester.pumpAndSettle();
      expect(
        _fieldText(tester, 'create-task-checksum-expected'),
        'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
      );

      // Task B form 1: null opts
      pending.set(CreateTask(req: Request(url: 'https://example.com/taskB1.zip')));
      router.go('/create');
      await tester.pumpAndSettle();
      expect(_fieldText(tester, 'create-task-checksum-expected'), '');

      // Task B form 2: empty Options
      pending.set(
        CreateTask(
          req: Request(url: 'https://example.com/taskB2.zip'),
          opts: Options(),
        ),
      );
      router.go('/create');
      await tester.pumpAndSettle();
      expect(_fieldText(tester, 'create-task-checksum-expected'), '');

      // Task B form 3: HTTP extra with no checksum
      pending.set(
        CreateTask(
          req: Request(url: 'https://example.com/taskB3.zip'),
          opts: Options(extra: OptsExtraHttp(connections: 8).toJson()),
        ),
      );
      router.go('/create');
      await tester.pumpAndSettle();
      expect(_fieldText(tester, 'create-task-checksum-expected'), '');

      // Submit task B3 and ensure no checksum is sent
      await tester.tap(find.byKey(const ValueKey('create-task-direct-download-toggle')));
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();

      expect(submitted?.opts?.extra, isNotNull);
      final extra = OptsExtraHttp.fromJson(Map<String, dynamic>.from(submitted!.opts!.extra! as Map));
      expect(extra.checksum, isNull);
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });

  testWidgets('checksum input validation blocks submission on invalid input (T3-UI)', (tester) async {
    tester.view.physicalSize = const Size(1024, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    CreateTask? submitted;
    final config = DownloaderConfig(downloadDir: '/downloads')..extra.defaultDirectDownload = true;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => config)
      ..bind(GopeedMethods.createTask, (task) {
        submitted = task;
        return 'task-id';
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());

    final container = ProviderContainer(
      overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
    );
    addTearDown(container.dispose);

    final pending = container.read(pendingCreateTaskProvider.notifier);
    pending.set(CreateTask(req: Request(url: 'https://example.com/file.zip')));

    final router = GoRouter(
      initialLocation: '/create',
      routes: [
        GoRoute(path: '/', builder: (_, _) => const SizedBox()),
        GoRoute(path: '/create', builder: (_, _) => const CreateTaskWindowPage()),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: shad.ShadcnApp.router(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          routerConfig: router,
          builder: (_, child) => AppComponentThemes(child: child!),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Expand advanced options
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();

    // Test 1: "abc" for SHA-256
    await tester.tap(find.text('SHA-256'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-checksum-expected')), 'abc');
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNull);

    // Test 2: 64 x "g" (non-hex) for SHA-256
    await tester.enterText(find.byKey(const ValueKey('create-task-checksum-expected')), 'g' * 64);
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNull);

    // Test 3: 63 x "a" (invalid length) for SHA-256
    await tester.enterText(find.byKey(const ValueKey('create-task-checksum-expected')), 'a' * 63);
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNull);

    // Test 4: 65 x "a" (invalid length) for SHA-256
    await tester.enterText(find.byKey(const ValueKey('create-task-checksum-expected')), 'a' * 65);
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNull);

    // Test 5: MD5 with sha256-length hash
    await tester.tap(find.text('MD5'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-checksum-expected')), 'a' * 64);
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNull);

    // Test 6: Valid MD5 in UPPER case
    submitted = null;
    await tester.tap(find.text('MD5'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const ValueKey('create-task-checksum-expected')),
      '0123456789ABCDEF0123456789ABCDEF',
    );
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNotNull);
    var extra = OptsExtraHttp.fromJson(Map<String, dynamic>.from(submitted!.opts!.extra! as Map));
    expect(extra.checksum?.algorithm, 'md5');
    expect(extra.checksum?.expected, '0123456789ABCDEF0123456789ABCDEF');

    // Test 7: Valid SHA-1 in lower case
    submitted = null;
    pending.set(CreateTask(req: Request(url: 'https://example.com/file2.zip')));
    router.go('/create');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('SHA-1'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const ValueKey('create-task-checksum-expected')),
      'da39a3ee5e6b4b0d3255bfef95601890afd80709',
    );
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNotNull);
    extra = OptsExtraHttp.fromJson(Map<String, dynamic>.from(submitted!.opts!.extra! as Map));
    expect(extra.checksum?.algorithm, 'sha1');
    expect(extra.checksum?.expected, 'da39a3ee5e6b4b0d3255bfef95601890afd80709');

    // Test 8: Valid SHA-256 in lower case
    submitted = null;
    pending.set(CreateTask(req: Request(url: 'https://example.com/file3.zip')));
    router.go('/create');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('SHA-256'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const ValueKey('create-task-checksum-expected')),
      'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    );
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNotNull);
    extra = OptsExtraHttp.fromJson(Map<String, dynamic>.from(submitted!.opts!.extra! as Map));
    expect(extra.checksum?.algorithm, 'sha256');
    expect(extra.checksum?.expected, 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855');

    // Test 9: Omitted checksum (empty hash)
    submitted = null;
    pending.set(CreateTask(req: Request(url: 'https://example.com/file4.zip')));
    router.go('/create');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-checksum-expected')), '   ');
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted, isNotNull);
    extra = OptsExtraHttp.fromJson(Map<String, dynamic>.from(submitted!.opts!.extra! as Map));
    expect(extra.checksum, isNull);
  });

  testWidgets('multi-URL and multi-file batch creation omits checksum (T4)', (tester) async {
    tester.view.physicalSize = const Size(1024, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final createdTasks = <CreateTask>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(GopeedMethods.createTask, (task) {
        createdTasks.add(task);
        return 'task-id';
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());

    final container = ProviderContainer(
      overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
    );
    addTearDown(container.dispose);

    final router = GoRouter(
      initialLocation: '/create',
      routes: [
        GoRoute(path: '/', builder: (_, _) => const SizedBox()),
        GoRoute(path: '/create', builder: (_, _) => const CreateTaskWindowPage()),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: shad.ShadcnApp.router(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          routerConfig: router,
          builder: (_, child) => AppComponentThemes(child: child!),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Enter 2 URLs
    await tester.enterText(
      find.byKey(const ValueKey('create-task-url-input')),
      'https://example.com/file1.zip\nhttps://example.com/file2.zip',
    );
    await tester.pumpAndSettle();

    // Advanced options should hide the checksum field
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('create-task-checksum-expected')), findsNothing);

    // Confirm creation
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();

    expect(createdTasks.length, 2);
    for (final t in createdTasks) {
      final extra = OptsExtraHttp.fromJson(Map<String, dynamic>.from(t.opts!.extra! as Map));
      expect(extra.checksum, isNull);
    }
  });

  testWidgets('checksum UI uses localized labels (T5)', (tester) async {
    tester.view.physicalSize = const Size(700, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'));

    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
        child: shad.ShadcnApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(
            child: CreateTaskWindowPage(
              initialTask: CreateTask(req: Request(url: 'https://example.com/file.zip')),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();

    expect(find.text('Checksum Algorithm'), findsOneWidget);
    expect(find.text('Checksum Hash'), findsOneWidget);
  });

  testWidgets('category shortcut highlights the current directory and toggles off when tapped again', (tester) async {
    final palette = await _pumpCreateTaskWithCategories(
      tester,
      downloadDir: 'C:/Downloads',
      categories: [
        DownloadCategory(name: 'Archives', path: 'E:/Archives'),
        DownloadCategory(name: 'Videos', path: 'F:/Videos'),
      ],
    );

    expect(_fieldText(tester, 'create-task-directory-input'), 'C:/Downloads');
    expect(_shortcutFill(tester, 0), palette.cardBg);
    expect(_shortcutFill(tester, 1), palette.cardBg);

    await tester.tap(find.byKey(const ValueKey('create-task-category-0')));
    await tester.pumpAndSettle();
    expect(_fieldText(tester, 'create-task-directory-input'), 'E:/Archives');
    expect(_shortcutFill(tester, 0), palette.brandSoft);
    expect(_shortcutFill(tester, 1), palette.cardBg);

    await tester.tap(find.byKey(const ValueKey('create-task-category-1')));
    await tester.pumpAndSettle();
    expect(_fieldText(tester, 'create-task-directory-input'), 'F:/Videos');
    expect(_shortcutFill(tester, 0), palette.cardBg);
    expect(_shortcutFill(tester, 1), palette.brandSoft);

    await tester.tap(find.byKey(const ValueKey('create-task-category-1')));
    await tester.pumpAndSettle();
    expect(_fieldText(tester, 'create-task-directory-input'), 'C:/Downloads');
    expect(_shortcutFill(tester, 0), palette.cardBg);
    expect(_shortcutFill(tester, 1), palette.cardBg);
    expect(tester.takeException(), isNull);
  });

  testWidgets('directory edits highlight a matching category once the debounce settles', (tester) async {
    final palette = await _pumpCreateTaskWithCategories(
      tester,
      downloadDir: 'C:/Downloads',
      categories: [
        DownloadCategory(name: 'Archives', path: 'E:/Archives'),
        DownloadCategory(name: 'Videos', path: 'F:/Videos'),
      ],
    );

    await tester.enterText(find.byKey(const ValueKey('create-task-directory-input')), 'F:/Videos');
    await tester.pump(const Duration(milliseconds: 100));
    expect(_shortcutFill(tester, 1), palette.cardBg);

    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    expect(_shortcutFill(tester, 0), palette.cardBg);
    expect(_shortcutFill(tester, 1), palette.brandSoft);

    await tester.enterText(find.byKey(const ValueKey('create-task-directory-input')), 'D:/Somewhere');
    await tester.pump(const Duration(milliseconds: 500));
    await tester.pumpAndSettle();
    expect(_shortcutFill(tester, 0), palette.cardBg);
    expect(_shortcutFill(tester, 1), palette.cardBg);
    expect(tester.takeException(), isNull);
  });

  testWidgets('clicking a shortcut selects its directory and skips auto categorize', (tester) async {
    CreateTask? submitted;
    final palette = await _pumpCreateTaskForSubmit(
      tester,
      categories: [
        DownloadCategory(name: 'Program', path: 'C:/Downloads/Program'),
        DownloadCategory(name: 'Archives', path: 'E:/Archives'),
      ],
      onCreate: (task) => submitted = task,
    );

    await tester.tap(find.byKey(const ValueKey('create-task-category-0')));
    await tester.pumpAndSettle();
    expect(_fieldText(tester, 'create-task-directory-input'), 'C:/Downloads/Program');
    expect(_shortcutFill(tester, 0), palette.brandSoft);
    expect(_shortcutFill(tester, 1), palette.cardBg);

    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pumpAndSettle();
    expect(submitted?.opts?.path, 'C:/Downloads/Program');
    expect(submitted?.opts?.skipAutoCategorize, isTrue);
    expect(tester.takeException(), isNull);
  });

  testWidgets('tapping the selected shortcut restores the default directory and still skips', (tester) async {
    CreateTask? submitted;
    final palette = await _pumpCreateTaskForSubmit(
      tester,
      categories: [
        DownloadCategory(name: 'Program', path: 'C:/Downloads/Program'),
        DownloadCategory(name: 'Archives', path: 'E:/Archives'),
      ],
      onCreate: (task) => submitted = task,
    );

    await tester.tap(find.byKey(const ValueKey('create-task-category-0')));
    await tester.pumpAndSettle();
    expect(_shortcutFill(tester, 0), palette.brandSoft);

    await tester.tap(find.byKey(const ValueKey('create-task-category-0')));
    await tester.pumpAndSettle();
    expect(_fieldText(tester, 'create-task-directory-input'), 'C:/Downloads');
    expect(_shortcutFill(tester, 0), palette.cardBg);
    expect(_shortcutFill(tester, 1), palette.cardBg);

    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pumpAndSettle();
    expect(submitted?.opts?.path, 'C:/Downloads');
    expect(submitted?.opts?.skipAutoCategorize, isTrue);
    expect(tester.takeException(), isNull);
  });

  testWidgets('the resolve dialog shows the suggested category directory', (tester) async {
    tester.view.physicalSize = const Size(700, 700);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(
        GopeedMethods.resolve,
        (_) => ResolveResult(
          id: 'resolved-id',
          res: Resource(files: [FileInfo(name: 'setup.exe', size: 1024)]),
          suggestedDownloadDir: '/downloads/Program',
        ),
      )
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(child: const CreateTaskWindowPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/setup.exe');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    final suggestion = find.byKey(const ValueKey('resolve-category-directory'));
    expect(suggestion, findsOneWidget);
    expect(tester.widget<Text>(suggestion).data, contains('/downloads/Program'));
    expect(tester.takeException(), isNull);
  });

  testWidgets('the default download directory toggle needs a manual directory choice', (tester) async {
    await _pumpCreateTaskWithCategories(
      tester,
      downloadDir: 'C:/Downloads',
      url: 'https://example.com/setup.exe',
      categories: [
        DownloadCategory(name: 'Program', path: 'C:/Downloads/Program', extensions: const ['exe']),
      ],
      onCreate: (task) {},
    );

    final checkbox = find.byKey(const ValueKey('create-task-remember-download-directory-checkbox'));
    // The global default directory is only saved for a directory the user
    // named, so the toggle stays disabled until then.
    expect(tester.widget<shad.Checkbox>(checkbox).onChanged, isNull);

    await tester.enterText(find.byKey(const ValueKey('create-task-directory-input')), 'D:/Other');
    await tester.pumpAndSettle();
    expect(tester.widget<shad.Checkbox>(checkbox).onChanged, isNotNull);
    await _settleToastTimers(tester);
  });

  testWidgets('moving the caret in the directory field does not skip auto categorize', (tester) async {
    CreateTask? submitted;
    await _pumpCreateTaskWithCategories(
      tester,
      downloadDir: 'C:/Downloads',
      url: 'https://example.com/setup.exe',
      categories: [
        DownloadCategory(name: 'Program', path: 'C:/Downloads/Program', extensions: const ['exe']),
      ],
      onCreate: (task) => submitted = task,
    );

    // Only focus the field and move the caret, the text stays untouched.
    final field = find.byKey(const ValueKey('create-task-directory-input'));
    await tester.tap(field, warnIfMissed: false);
    await tester.pumpAndSettle();
    await tester.tapAt(tester.getTopLeft(field) + const Offset(30, 10));
    await tester.pumpAndSettle();

    expect(_fieldText(tester, 'create-task-directory-input'), 'C:/Downloads');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    expect(submitted?.opts?.path, 'C:/Downloads');
    expect(submitted?.opts?.skipAutoCategorize, isFalse);
    await _settleToastTimers(tester);
  });

  testWidgets('the resolve dialog shows the directory the user named, not the suggestion', (tester) async {
    tester.view.physicalSize = const Size(700, 700);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(
        GopeedMethods.resolve,
        (_) => ResolveResult(
          id: 'resolved-id',
          res: Resource(files: [FileInfo(name: 'setup.exe', size: 1024)]),
          suggestedDownloadDir: '/downloads/Program',
        ),
      )
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(child: const CreateTaskWindowPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-directory-input')), '/work');
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/setup.exe');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    final shown = find.byKey(const ValueKey('resolve-category-directory'));
    expect(shown, findsOneWidget);
    expect(tester.widget<Text>(shown).data, contains('/work'));
    expect(tester.widget<Text>(shown).data, isNot(contains('/downloads/Program')));
    expect(tester.takeException(), isNull);
  });

  testWidgets('a resolved file list is split into tasks that keep the directory structure', (tester) async {
    tester.view.physicalSize = const Size(700, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    CreateTaskBatch? batch;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(
        GopeedMethods.resolve,
        (_) => ResolveResult(
          res: Resource(
            name: 'bundle',
            files: [
              FileInfo(name: 'first.zip', path: 'first.zip', size: 1),
              FileInfo(name: 'second.zip', path: 'second.zip', size: 1),
            ],
          ),
        ),
      )
      ..bind(GopeedMethods.createTaskBatch, (request) {
        batch = request;
        return ['task-1', 'task-2'];
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(child: const CreateTaskWindowPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/bundle');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pumpAndSettle();

    expect(batch, isNotNull);
    expect(batch!.reqs, hasLength(2));
    for (final item in batch!.reqs!) {
      // The split tasks keep the structure of the resolved resource, so they
      // are never classified into a category.
      expect(item.opts?.skipAutoCategorize, isTrue);
    }
    expect(batch!.opts?.skipAutoCategorize, isTrue);
    expect(tester.takeException(), isNull);
    await _settleToastTimers(tester);
  });

  testWidgets('a single selected file of a multi file resource is still not classified', (tester) async {
    tester.view.physicalSize = const Size(700, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    CreateTaskBatch? batch;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(
        GopeedMethods.resolve,
        (_) => ResolveResult(
          res: Resource(
            files: [
              FileInfo(name: 'first.zip', path: 'first.zip', size: 1),
              FileInfo(name: 'second.zip', path: 'second.zip', size: 1),
            ],
          ),
        ),
      )
      ..bind(GopeedMethods.createTaskBatch, (request) {
        batch = request;
        return ['task-1'];
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(child: const CreateTaskWindowPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/bundle');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    // Only the first file stays selected, the resource itself is a multi file
    // resource and keeps its structure.
    // Deselect the second file, the resource itself stays a multi file
    // resource.
    final treeCheckboxes = find.descendant(of: find.byType(ResolveFileTree), matching: find.byType(shad.Checkbox));
    await tester.tap(treeCheckboxes.last);
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pumpAndSettle();

    expect(batch, isNotNull);
    expect(batch!.reqs, hasLength(1));
    for (final item in batch!.reqs!) {
      expect(item.opts?.skipAutoCategorize, isTrue);
    }
    expect(batch!.opts?.skipAutoCategorize, isTrue);
    expect(tester.takeException(), isNull);
    await _settleToastTimers(tester);
  });

  group('FTP links', _ftpTests);
}

Future<void> _pumpCreateRoute(WidgetTester tester, CapabilityRegistry registry, {CreateTask? initialTask}) async {
  tester.view.physicalSize = const Size(1024, 900);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  final router = GoRouter(
    initialLocation: '/create',
    routes: [
      GoRoute(path: '/', builder: (_, _) => const SizedBox()),
      GoRoute(
        path: '/create',
        builder: (_, _) => CreateTaskWindowPage(initialTask: initialTask),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
      child: shad.ShadcnApp.router(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        builder: (context, child) => AppComponentThemes(child: child ?? const SizedBox.shrink()),
        routerConfig: router,
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void _ftpTests() {
  const urlInput = ValueKey('create-task-url-input');
  const connectionsInput = ValueKey('create-task-connections-input');
  const tlsSelect = ValueKey('create-task-ftp-tls');

  testWidgets('ftp links offer the TLS mode and carry FTP options through resolve', (tester) async {
    ResolveTask? resolved;
    CreateTask? submitted;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(GopeedMethods.resolve, (task) {
        resolved = task;
        return ResolveResult(
          id: 'ftp-resolved',
          res: Resource(
            name: 'big.iso',
            files: [FileInfo(name: 'big.iso', size: 1024)],
          ),
        );
      })
      ..bind(GopeedMethods.createTask, (task) {
        submitted = task;
        return 'ftp-task';
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    await _pumpCreateRoute(tester, registry);
    expect(find.byKey(tlsSelect), findsNothing);

    // ftps:// and ftpes:// fix the TLS mode, so only ftp:// offers the select.
    for (final url in ['ftps://files.example/pub/big.iso', 'ftpes://files.example/pub/big.iso']) {
      await tester.enterText(find.byKey(urlInput), url);
      await tester.pump();
      expect(find.byKey(tlsSelect), findsNothing, reason: url);
    }
    await tester.enterText(find.byKey(urlInput), 'FTP://files.example/pub/big.iso');
    await tester.pump();
    expect(find.byKey(tlsSelect), findsOneWidget);
    expect(find.descendant(of: find.byKey(tlsSelect), matching: find.text('None')), findsOneWidget);

    await tester.tap(find.byKey(tlsSelect));
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('create-task-ftp-tls-implicit')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('create-task-ftp-tls-explicit')));
    await tester.pumpAndSettle();
    expect(find.descendant(of: find.byKey(tlsSelect), matching: find.text('Explicit TLS (FTPES)')), findsOneWidget);

    await tester.enterText(find.byKey(connectionsInput), '64');
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    // The confirm button spins while the resolve dialog is open, so the frames never settle.
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pumpAndSettle();

    // The TLS mode travels on the request, so the resolve already uses it.
    expect(resolved?.req?.url, 'FTP://files.example/pub/big.iso');
    expect(resolved?.req?.extra, {'tls': 'explicit'});
    expect(submitted?.rid, 'ftp-resolved');
    final createOptions = submitted?.opts?.extra! as Map<String, dynamic>;
    expect(createOptions['connections'], 32);
    expect(createOptions.containsKey('tls'), isFalse);
  });

  testWidgets('an ftp .torrent link restores its TLS mode and is not treated as BitTorrent', (tester) async {
    CreateTask? submitted;
    final config = DownloaderConfig(downloadDir: '/downloads')..extra.defaultDirectDownload = true;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => config)
      ..bind(GopeedMethods.createTask, (task) {
        submitted = task;
        return 'ftp-task';
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    await _pumpCreateRoute(
      tester,
      registry,
      initialTask: CreateTask(
        req: Request(
          url: 'ftp://files.example/pub/linux.torrent',
          extra: ReqExtraFtp(tls: 'implicit').toJson(),
        ),
        opts: Options(extra: OptsExtraFtp(connections: 6).toJson()),
      ),
    );

    expect(_fieldText(tester, 'create-task-connections-input'), '6');
    expect(find.descendant(of: find.byKey(tlsSelect), matching: find.text('Implicit TLS (FTPS)')), findsOneWidget);
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    // The HTTP tab holds the certificate and post-download switches FTP uses; trackers stay hidden.
    expect(find.byKey(const ValueKey('create-task-skip-verify-cert')), findsOneWidget);
    expect(find.text('Trackers'), findsNothing);
    // Checksum verification is HTTP only, so an FTP link does not offer it.
    expect(find.byKey(const ValueKey('create-task-checksum-expected')), findsNothing);

    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(submitted?.req?.url, 'ftp://files.example/pub/linux.torrent');
    expect(submitted?.req?.extra, {'tls': 'implicit'});
    final options = OptsExtraFtp.fromJson(submitted?.opts?.extra! as Map<String, dynamic>);
    expect(options.connections, 6);
  });

  testWidgets('mixed links cap only the FTP task at 32 connections', (tester) async {
    final submitted = <CreateTask>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(GopeedMethods.createTask, (task) {
        submitted.add(task);
        return 'task-${submitted.length}';
      })
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
    await _pumpCreateRoute(tester, registry);

    await tester.enterText(find.byKey(urlInput), 'https://example.com/a.zip\nftpes://files.example/b.iso');
    await tester.pump();
    expect(find.byKey(tlsSelect), findsNothing);
    await tester.enterText(find.byKey(connectionsInput), '64');
    await tester.pump();
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();

    final byUrl = {for (final task in submitted) task.req!.url: task.opts!.extra! as Map<String, dynamic>};
    expect(byUrl.keys, unorderedEquals(['https://example.com/a.zip', 'ftpes://files.example/b.iso']));
    expect(byUrl['https://example.com/a.zip']!['connections'], 64);
    expect(byUrl['https://example.com/a.zip']!.containsKey('tls'), isFalse);
    expect(byUrl['ftpes://files.example/b.iso']!['connections'], 32);
    expect(byUrl['ftpes://files.example/b.iso']!.containsKey('tls'), isFalse);
    expect(submitted.map((task) => task.req!.extra), everyElement(isNull));
  });
}

/// Unmounts the tree and lets the shadcn toast close timers fire, so a test
/// ends without pending timers.
Future<void> _settleToastTimers(WidgetTester tester) async {
  await tester.pumpWidget(const SizedBox());
  await tester.pump(const Duration(seconds: 6));
}

/// Pumps the create dialog behind a router, so a finished submission can
/// navigate away, and captures the CreateTask that reaches the backend.
Future<AppPalette> _pumpCreateTaskForSubmit(
  WidgetTester tester, {
  required void Function(CreateTask task) onCreate,
  String downloadDir = 'C:/Downloads',
  List<DownloadCategory> categories = const [],
}) async {
  tester.view.physicalSize = const Size(1024, 720);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  DownloadDirectoryPicker.debugPlatformOverride = TargetPlatform.windows;
  addTearDown(() => DownloadDirectoryPicker.debugPlatformOverride = null);

  final config = DownloaderConfig(downloadDir: downloadDir)
    ..extra.defaultDirectDownload = true
    ..categories = categories;
  final registry = CapabilityRegistry(createAppCapabilityCodecs())
    ..bind(GopeedMethods.getConfig, (_) => config)
    ..bind(GopeedMethods.createTask, (task) {
      onCreate(task);
      return 'created-task';
    })
    ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
  final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

  final router = GoRouter(
    initialLocation: '/create',
    routes: [
      GoRoute(path: '/', builder: (_, _) => const SizedBox()),
      GoRoute(path: '/create', builder: (_, _) => const CreateTaskWindowPage()),
    ],
  );
  addTearDown(router.dispose);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
      child: shad.ShadcnApp.router(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        builder: (context, child) => AppComponentThemes(child: child ?? const SizedBox.shrink()),
        routerConfig: router,
      ),
    ),
  );
  await tester.pumpAndSettle();
  await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/setup.exe');
  await tester.pumpAndSettle();
  return AppPalette.of(tester.element(find.byType(CreateTaskWindowPage)));
}

Future<AppPalette> _pumpCreateTaskWithCategories(
  WidgetTester tester, {
  required String downloadDir,
  required List<DownloadCategory> categories,
  String? url,
  String? rename,
  void Function(CreateTask task)? onCreate,
}) async {
  tester.view.physicalSize = const Size(1024, 720);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  DownloadDirectoryPicker.debugPlatformOverride = TargetPlatform.windows;
  addTearDown(() => DownloadDirectoryPicker.debugPlatformOverride = null);

  final config = DownloaderConfig(downloadDir: downloadDir)
    ..extra.defaultDirectDownload = onCreate != null
    ..categories = categories;
  final registry = CapabilityRegistry(createAppCapabilityCodecs())
    ..bind(GopeedMethods.getConfig, (_) => config)
    ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
  if (onCreate != null) {
    registry.bind(GopeedMethods.createTask, (task) {
      onCreate(task);
      return 'created-task';
    });
  }
  final capabilities = AppCapabilities(LocalCapabilityInvoker(registry));

  await tester.pumpWidget(
    ProviderScope(
      overrides: [appCapabilitiesProvider.overrideWithValue(capabilities)],
      child: shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: AppComponentThemes(child: const CreateTaskWindowPage()),
      ),
    ),
  );
  await tester.pumpAndSettle();
  if (url != null) {
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), url);
    await tester.pumpAndSettle();
  }
  if (rename != null) {
    await tester.enterText(find.byKey(const ValueKey('create-task-rename-input')), rename);
    await tester.pumpAndSettle();
  }
  return AppPalette.of(tester.element(find.byType(CreateTaskWindowPage)));
}

// Asserts through the button's AnimatedContainer decoration, so a restructure of the button fails loudly.
Color? _shortcutFill(WidgetTester tester, int index) {
  final container = tester.widget<AnimatedContainer>(
    find.descendant(of: find.byKey(ValueKey('create-task-category-$index')), matching: find.byType(AnimatedContainer)),
  );
  return (container.decoration! as BoxDecoration).color;
}

String _fieldText(WidgetTester tester, String key) {
  final editable = find.descendant(of: find.byKey(ValueKey(key)), matching: find.byType(EditableText));
  return tester.widget<EditableText>(editable).controller.text;
}
