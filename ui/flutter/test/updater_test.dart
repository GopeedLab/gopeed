import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/settings/presentation/widgets/app_update_dialog.dart';
import 'package:gopeed/features/settings/presentation/widgets/app_update_notes_viewport.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';
import 'package:gopeed/util/arch/arch.dart';
import 'package:gopeed/util/package_info.dart' as app_package;
import 'package:gopeed/util/updater.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  TestWidgetsFlutterBinding.ensureInitialized().platformDispatcher.localeTestValue = const Locale('en');

  setUpAll(() {
    app_package.packageInfo = PackageInfo(
      appName: 'Gopeed',
      packageName: 'com.gopeed.Gopeed',
      version: '1.7.0',
      buildNumber: '1',
    );
  });

  test('update checks pass the client target and retain backend download candidates', () async {
    final transport = _UpdateTransport();
    api.setTransportForTesting(transport);
    final info = await checkUpdate(service: LocalAppCapabilities.instance.capabilities.gopeed);
    expect(transport.requests, 1);
    expect(transport.path, 'api/v1/app/updates/check');
    expect(transport.data, {
      'currentVersion': app_package.appVersion,
      'channel': updateChannel?.name ?? '',
      'architecture': getArch().name,
    });
    expect(info!.version, '2.0.0');
    expect(info.assetName, 'Gopeed-v2.0.0-macos-arm64.dmg');
    expect(info.downloadUrls, ['https://example.com/package.dmg', 'https://mirror.example/package.dmg']);
    transport.result = null;
    expect(await checkUpdate(service: LocalAppCapabilities.instance.capabilities.gopeed), isNull);
  });

  test('selects the matching bilingual release notes', () {
    const notes = '# Release notes\n\n- English change\n\n# 更新日志\n\n- 中文更新';
    expect(localizedReleaseNotes(notes, 'en'), '# Release notes\n\n- English change');
    expect(localizedReleaseNotes(notes, 'zh-CN'), '# 更新日志\n\n- 中文更新');
  });

  testWidgets('update dialog shows beta versions without tag prefixes', (tester) async {
    tester.view.physicalSize = const Size(390, 760);
    tester.view.devicePixelRatio = 1;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    app_package.packageInfo = PackageInfo(
      appName: 'Gopeed',
      packageName: 'com.gopeed.Gopeed',
      version: '2.0.0.1',
      buildNumber: '1',
    );
    addTearDown(() {
      app_package.packageInfo = PackageInfo(
        appName: 'Gopeed',
        packageName: 'com.gopeed.Gopeed',
        version: '1.7.0',
        buildNumber: '1',
      );
    });

    const version = VersionInfo(
      version: '2.0.0-beta.2',
      changeLog: '# Release notes\n\n- Faster downloads\n\n# 更新日志\n\n- 下载更快',
      releaseUrl: 'https://example.com/release',
      assetName: '',
      downloadUrls: [],
    );
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Builder(
          builder: (context) => shad.SecondaryButton(
            onPressed: () => showAppUpdateDialog(context, versionInfo: version, onUpdate: (_, _) async {}),
            child: const Text('Open update'),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open update'));
    await tester.pumpAndSettle();

    expect(find.byKey(const ValueKey('app-update-dialog')), findsOneWidget);
    expect(find.text('A new version is available'), findsOneWidget);
    expect(find.text('2.0.0-beta.1  →  2.0.0-beta.2'), findsOneWidget);
    expect(find.textContaining('v2.0.0'), findsNothing);
    expect(find.text('Faster downloads'), findsOneWidget);
    expect(find.text('下载更快'), findsNothing);
    expect(find.text('Later'), findsOneWidget);
    expect(find.text('Update'), findsOneWidget);
    expect(find.text("What's New"), findsNothing);
    final versionText = tester.widget<Text>(find.text('2.0.0-beta.1  →  2.0.0-beta.2'));
    expect(versionText.style?.decoration, TextDecoration.none);
    expect(versionText.style?.inherit, isFalse);
    expect(find.byIcon(Icons.system_update_alt_outlined), findsNothing);
    final dialog = tester.getRect(find.byKey(const ValueKey('app-update-dialog')));
    expect(dialog.width, lessThan(320));
    expect(tester.getCenter(find.byKey(const ValueKey('app-update-heading'))).dx, closeTo(dialog.center.dx, 0.01));
    expect(tester.getCenter(find.byKey(const ValueKey('app-update-actions'))).dx, closeTo(dialog.center.dx, 0.01));
    expect(tester.takeException(), isNull);
  });

  testWidgets('release notes keep the last line above the fade when scrolled to the end', (tester) async {
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.dark(),
        materialTheme: AppTheme.materialDark(),
        home: Center(
          child: SizedBox(
            width: 280,
            child: AppUpdateNotesViewport(
              child: Column(children: [for (var i = 0; i < 30; i++) Text('Release note $i')]),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final scrollbar = tester.widget<shad.Scrollbar>(find.byType(shad.Scrollbar));
    expect(scrollbar.thumbVisibility, isTrue);
    final controller = scrollbar.controller!;
    expect(controller.position.maxScrollExtent, greaterThan(0));
    controller.jumpTo(controller.position.maxScrollExtent);
    await tester.pumpAndSettle();
    final viewport = tester.getRect(find.byType(AppUpdateNotesViewport));
    final lastLine = tester.getRect(find.text('Release note 29'));
    expect(lastLine.bottom, lessThanOrEqualTo(viewport.bottom - 24));
    expect(lastLine.top, greaterThan(viewport.top));
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });
}

class _UpdateTransport implements GopeedTransport {
  int requests = 0;
  String? path;
  dynamic data;
  Map<String, dynamic>? result = {
    'version': '2.0.0',
    'changeLog': 'Release notes',
    'releaseUrl': 'https://example.com/release',
    'assetName': 'Gopeed-v2.0.0-macos-arm64.dmg',
    'downloadUrls': ['https://example.com/package.dmg', 'https://mirror.example/package.dmg'],
  };
  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    requests++;
    this.path = path;
    this.data = data;
    return {'code': 0, 'data': result};
  }

  @override
  String join(String path) => path;
}
