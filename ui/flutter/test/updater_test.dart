import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/features/settings/presentation/widgets/app_update_dialog.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/util/arch/arch.dart';
import 'package:gopeed/util/github_mirror.dart';
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

  group('version comparison', () {
    test('uses semantic version precedence', () {
      expect(isNewerVersion('1.10.0', '1.9.9'), isTrue);
      expect(isNewerVersion('2.0.0-beta.2', '2.0.0-beta.1'), isTrue);
      expect(isNewerVersion('2.0.0-rc.1', '2.0.0-beta.2'), isTrue);
      expect(isNewerVersion('2.0.0', '2.0.0-rc.1'), isTrue);
      expect(isNewerVersion('1.2.3', '1.2.3'), isFalse);
      expect(isNewerVersion('1.2.2', '1.2.3'), isFalse);
      expect(isNewerVersion('invalid', '1.2.3'), isFalse);
    });
  });

  group('release selection', () {
    Map<String, dynamic> release(String version, {bool prerelease = false, bool draft = false}) => {
      'tag_name': 'v$version',
      'prerelease': prerelease,
      'draft': draft,
      'body': '$version notes',
      'html_url': 'https://example.com/$version',
    };

    test('stable builds ignore preview releases', () {
      final selected = selectUpdateRelease([release('2.1.0-beta.1', prerelease: true), release('2.0.1')], '2.0.0');

      expect(selected?.version, '2.0.1');
    });

    test('preview builds advance within the same preview line', () {
      final selected = selectUpdateRelease([
        release('2.0.0-beta.2', prerelease: true),
        release('1.9.9'),
        release('2.1.0-beta.1', prerelease: true),
      ], '2.0.0-beta.1');

      expect(selected?.version, '2.0.0-beta.2');
    });

    test('four-part platform beta selects the next GitHub prerelease and matching asset', () {
      final currentVersion = app_package.normalizeAppVersion('2.0.0.1');
      final selected = selectUpdateRelease([release('2.0.0-beta.2', prerelease: true)], currentVersion);

      expect(selected?.version, '2.0.0-beta.2');
      expect(
        updateAssetName(selected!.version, channel: UpdateChannel.macosDmg, architecture: Architecture.arm64),
        'Gopeed-v2.0.0-beta.2-macos-arm64.dmg',
      );
    });

    test('preview builds graduate to the final stable release', () {
      final selected = selectUpdateRelease([
        release('2.0.0-beta.2', prerelease: true),
        release('2.0.0'),
      ], '2.0.0-beta.1');

      expect(selected?.version, '2.0.0');
    });

    test('selects the greatest eligible version and ignores invalid releases', () {
      final selected = selectUpdateRelease([
        release('2.0.0-beta.3', prerelease: true),
        release('2.0.0-beta.4', prerelease: true, draft: true),
        {'tag_name': 'not-a-version'},
        release('2.0.0-beta.2', prerelease: true),
      ], '2.0.0-beta.1');

      expect(selected?.version, '2.0.0-beta.3');
    });
  });

  group('release assets', () {
    test('uses the release naming convention for desktop packages', () {
      expect(
        updateAssetName('1.8.0', channel: UpdateChannel.windowsPortable, architecture: Architecture.x64),
        'Gopeed-v1.8.0-windows-amd64-portable.zip',
      );
      expect(
        updateAssetName('1.8.0', channel: UpdateChannel.macosDmg, architecture: Architecture.arm64),
        'Gopeed-v1.8.0-macos-arm64.dmg',
      );
      expect(
        updateAssetName('1.8.0', channel: UpdateChannel.linuxDeb, architecture: Architecture.ia32),
        'Gopeed-v1.8.0-linux-386.deb',
      );
    });

    test('maps Android ABI names', () {
      expect(
        updateAssetName('1.8.0', channel: UpdateChannel.androidApk, architecture: Architecture.arm),
        'Gopeed-v1.8.0-android-armeabi-v7a.apk',
      );
      expect(
        updateAssetName('1.8.0', channel: UpdateChannel.androidApk, architecture: Architecture.arm64),
        'Gopeed-v1.8.0-android-arm64-v8a.apk',
      );
    });
  });

  test('selects the matching bilingual release notes', () {
    const notes = '# Release notes\n\n- English change\n\n# 更新日志\n\n- 中文更新';
    expect(localizedReleaseNotes(notes, 'en'), '# Release notes\n\n- English change');
    expect(localizedReleaseNotes(notes, 'zh-CN'), '# 更新日志\n\n- 中文更新');
  });

  test('builds enabled GitHub proxy URLs and ignores deleted mirrors', () {
    final config = ExtraConfigGithubMirror(
      mirrors: [
        GithubMirror(type: GithubMirrorType.ghProxy, url: 'https://mirror.example'),
        GithubMirror(type: GithubMirrorType.ghProxy, url: 'https://deleted.example', isDeleted: true),
      ],
    );
    expect(
      githubMirrorUrls(
        'https://github.com/GopeedLab/gopeed/releases/download/v1.8.0/app.zip',
        MirrorType.githubRelease,
        config: config,
      ),
      ['https://mirror.example/https://github.com/GopeedLab/gopeed/releases/download/v1.8.0/app.zip'],
    );
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
    expect(find.text('Update Now'), findsOneWidget);
    final dialog = tester.getRect(find.byKey(const ValueKey('app-update-dialog')));
    expect(dialog.width, lessThan(320));
    expect(tester.getCenter(find.byKey(const ValueKey('app-update-heading'))).dx, closeTo(dialog.center.dx, 0.01));
    expect(tester.getCenter(find.byKey(const ValueKey('app-update-actions'))).dx, closeTo(dialog.center.dx, 0.01));
    expect(tester.takeException(), isNull);
  });
}
