import 'package:flutter/widgets.dart';
import 'package:flutter/widgets.dart' as flutter show Row;
import 'package:flutter/material.dart' show Icons;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/capabilities/storage_capability.dart';
import 'package:gopeed/features/tasks/presentation/pages/create_task_window_page.dart';
import 'package:gopeed/shared/services/download_directory_picker.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_design_tokens.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('browser extension request parameters populate the form and survive submission', (
    WidgetTester tester,
  ) async {
    tester.view.physicalSize = const Size(760, 720);
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
      ..extra.downloadCategories = [
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
    final directoryComponent = find.byKey(const ValueKey('create-task-directory-component'));
    final directoryOptionsRow = find.byKey(const ValueKey('create-task-directory-options-row'));
    final directoryOptionsLayout = find.byKey(const ValueKey('create-task-directory-options-layout'));
    final directoryPicker = find.byKey(const ValueKey('create-task-directory-picker'));
    expect(tester.widget<shad.Checkbox>(rememberDirectory).state, shad.CheckboxState.unchecked);
    expect(tester.widget<shad.Checkbox>(rememberDirectory).size, isNull);
    expect(tester.getSize(rememberDirectory), const Size.square(AppDesignTokens.checkboxSize));
    expect(tester.getCenter(rememberDirectory).dx, closeTo(tester.getCenter(directDownloadCheckbox).dx, 0.5));
    expect(tester.widget<shad.IconButton>(directoryPicker).variance, same(shad.ButtonVariance.outline));
    expect(tester.widget<flutter.Row>(directoryOptionsLayout).mainAxisAlignment, MainAxisAlignment.spaceBetween);
    expect(tester.getCenter(rememberDirectoryRow).dy, closeTo(tester.getCenter(shortcutRow).dy, 0.5));
    expect(tester.getTopRight(directoryOptionsRow).dx, closeTo(tester.getTopRight(directoryComponent).dx, 0.5));
    expect(
      tester.getTopRight(find.byKey(const ValueKey('create-task-category-3'))).dx,
      closeTo(tester.getTopRight(directoryComponent).dx, 0.5),
    );

    await tester.tap(directoryPicker);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('android-app-directory-option')));
    await tester.pumpAndSettle();
    expect(tester.widget<shad.Checkbox>(rememberDirectory).state, shad.CheckboxState.unchecked);

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
    final options = OptsExtraHttp.fromJson(submitted?.opts?.extra! as Map<String, dynamic>);
    expect(options.connections, 12);
    expect(options.autoTorrent, isTrue);
    expect(options.deleteTorrentAfterDownload, isTrue);
    expect(options.autoExtract, isTrue);
    expect(options.archivePassword, 'archive-password');
    expect(options.deleteAfterExtract, isTrue);
    expect(submitted?.opts?.asDefaultPath, isTrue);
  });
}

String _fieldText(WidgetTester tester, String key) {
  final editable = find.descendant(of: find.byKey(ValueKey(key)), matching: find.byType(EditableText));
  return tester.widget<EditableText>(editable).controller.text;
}
