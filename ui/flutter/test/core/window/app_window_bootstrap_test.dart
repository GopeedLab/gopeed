import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/core/window/app_window_bootstrap.dart';
import 'package:gopeed/core/window/app_window_payload.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const windowChannel = MethodChannel('window_manager');
  const screenChannel = MethodChannel('dev.leanflutter.plugins/screen_retriever');
  const display = {
    'id': 'primary',
    'size': {'width': 1920.0, 'height': 1080.0},
    'visiblePosition': {'dx': 0.0, 'dy': 0.0},
  };
  final messenger = TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  final calls = <MethodCall>[];

  setUp(() {
    calls.clear();
    messenger.setMockMethodCallHandler(windowChannel, (call) async {
      calls.add(call);
      if (call.method == 'setHasShadow' && defaultTargetPlatform == TargetPlatform.linux) {
        throw MissingPluginException('setHasShadow is not supported on Linux');
      }
      return switch (call.method) {
        'isFullScreen' || 'isMaximized' || 'isMinimized' => false,
        'getBounds' => {'x': 0.0, 'y': 0.0, 'width': 820.0, 'height': 620.0},
        _ => null,
      };
    });
    messenger.setMockMethodCallHandler(screenChannel, (call) async {
      return switch (call.method) {
        'getPrimaryDisplay' => display,
        'getAllDisplays' => {
          'displays': [display],
        },
        'getCursorScreenPoint' => {'dx': 0.0, 'dy': 0.0},
        _ => null,
      };
    });
  });

  tearDown(() {
    messenger.setMockMethodCallHandler(windowChannel, null);
    messenger.setMockMethodCallHandler(screenChannel, null);
  });

  void expectPlatformShadow(TargetPlatform platform) {
    final shadows = calls.where((call) => call.method == 'setHasShadow').map((call) => call.arguments);
    expect(
      shadows,
      platform == TargetPlatform.linux
          ? isEmpty
          : [
              {'hasShadow': true},
            ],
    );
  }

  testWidgets('hidden Windows startup does not restore maximized state', (tester) async {
    final saved = WindowStateConfig(isMaximized: true, width: 1200, height: 800);
    await AppWindowBootstrap.setupMainWindow(hidden: true);
    await tester.pump();
    calls.clear();

    await AppWindowBootstrap.restoreMainWindowState(saved, hidden: true);

    expect(calls.map((call) => call.method), contains('setBounds'));
    expect(calls.map((call) => call.method).where((method) => ['maximize', 'show', 'focus'].contains(method)), isEmpty);
    expect(saved.isMaximized, isTrue);
  }, variant: TargetPlatformVariant.only(TargetPlatform.windows));

  testWidgets('normal Windows startup restores maximized state', (tester) async {
    await AppWindowBootstrap.restoreMainWindowState(WindowStateConfig(isMaximized: true), hidden: false);

    expect(calls.map((call) => call.method), contains('maximize'));
  }, variant: TargetPlatformVariant.only(TargetPlatform.windows));

  for (final platform in [TargetPlatform.linux, TargetPlatform.windows, TargetPlatform.macOS]) {
    for (final subWindow in [false, true]) {
      testWidgets(
        '${platform.name} ${subWindow ? 'child' : 'main'} window shows with supported shadow behavior',
        (tester) async {
          if (subWindow) {
            await AppWindowBootstrap.setupSubWindow(AppWindowType.createTask, localeConfig: 'en');
          } else {
            await AppWindowBootstrap.setupMainWindow();
          }
          await tester.pump();
          if (subWindow && platform == TargetPlatform.windows) {
            await tester.pump(const Duration(milliseconds: 180));
          }

          expectPlatformShadow(platform);
          expect(calls.map((call) => call.method), containsAllInOrder(['show', 'focus']));
        },
        variant: TargetPlatformVariant.only(platform),
      );
    }

    testWidgets('${platform.name} hidden main window stays hidden', (tester) async {
      await AppWindowBootstrap.setupMainWindow(hidden: true);
      await tester.pump();

      expectPlatformShadow(platform);
      expect(calls.where((call) => call.method == 'show' || call.method == 'focus'), isEmpty);
    }, variant: TargetPlatformVariant.only(platform));
  }
}
