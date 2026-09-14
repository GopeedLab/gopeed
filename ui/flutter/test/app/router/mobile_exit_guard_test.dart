import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gopeed/app/router/mobile_exit_guard.dart';
import 'package:gopeed/shared/navigation/app_exit_confirmation_controller.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_toast.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  late List<MethodCall> exits;
  late GoRouter router;

  setUp(() {
    exits = [];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'SystemNavigator.pop') exits.add(call);
        return null;
      },
    );
    router = GoRouter(
      routes: [
        ShellRoute(
          builder: (_, _, child) => MobileExitGuard(child: child),
          routes: [
            GoRoute(
              path: '/',
              builder: (_, _) => const SizedBox.expand(),
              routes: [
                GoRoute(path: 'details', builder: (_, _) => const SizedBox.expand()),
                for (final path in ['extensions', 'settings'])
                  GoRoute(path: path, builder: (_, _) => const SizedBox.expand()),
              ],
            ),
          ],
        ),
      ],
    );
  });

  tearDown(() {
    router.dispose();
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      null,
    );
  });

  Future<void> mount(WidgetTester tester) async {
    await tester.pumpWidget(
      shad.ShadcnApp.router(theme: AppTheme.dark(), materialTheme: AppTheme.materialDark(), routerConfig: router),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('root back shows bottom prompt, second back exits', (tester) async {
    await mount(tester);
    await tester.binding.handlePopRoute();
    await tester.pump();
    expect(exits, isEmpty);
    expect(find.byType(AppToastContent), findsOneWidget);
    expect(tester.getCenter(find.byType(AppToastContent)).dx, 400);
    expect(tester.getBottomLeft(find.byType(AppToastContent)).dy, 576);
    await tester.pump(const Duration(milliseconds: 1900));
    await tester.binding.handlePopRoute();
    await tester.pump();
    expect(exits, hasLength(1));
    expect(find.byType(AppToastContent), findsNothing);
  }, variant: TargetPlatformVariant.only(TargetPlatform.android));

  testWidgets('expired prompt requires a fresh pair of back actions', (tester) async {
    await mount(tester);
    await tester.binding.handlePopRoute();
    await tester.pump();
    await tester.pump(AppExitConfirmationController.confirmationWindow);
    expect(find.byType(AppToastContent), findsNothing);
    await tester.binding.handlePopRoute();
    await tester.pump();
    expect(exits, isEmpty);
    expect(find.byType(AppToastContent), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
  }, variant: TargetPlatformVariant.only(TargetPlatform.android));

  testWidgets('nested page pops without prompting or exiting', (tester) async {
    await mount(tester);
    router.push('/details');
    await tester.pumpAndSettle();
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(router.routeInformationProvider.value.uri.path, '/');
    expect(find.byType(AppToastContent), findsNothing);
    expect(exits, isEmpty);
  }, variant: TargetPlatformVariant.only(TargetPlatform.android));
  testWidgets('backgrounding clears the exit confirmation', (tester) async {
    await mount(tester);
    await tester.binding.handlePopRoute();
    await tester.pump();
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pump();
    expect(find.byType(AppToastContent), findsNothing);
    await tester.binding.handlePopRoute();
    await tester.pump();
    expect(exits, isEmpty);
    expect(find.byType(AppToastContent), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
  }, variant: TargetPlatformVariant.only(TargetPlatform.android));
  for (final path in ['/extensions', '/settings']) {
    testWidgets('$path returns to tasks before confirming exit', (tester) async {
      await mount(tester);
      router.go(path);
      await tester.pumpAndSettle();
      expect(router.canPop(), isTrue);
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(router.routeInformationProvider.value.uri.path, '/');
      expect(find.byType(AppToastContent), findsNothing);
      expect(exits, isEmpty);
      await tester.binding.handlePopRoute();
      await tester.pump();
      expect(find.byType(AppToastContent), findsOneWidget);
      expect(exits, isEmpty);
      await tester.binding.handlePopRoute();
      await tester.pump();
      expect(exits, hasLength(1));
    }, variant: TargetPlatformVariant.only(TargetPlatform.android));
  }
  testWidgets(
    'dialog back dismisses the dialog before exit confirmation',
    (tester) async {
      await mount(tester);
      final context = router.routerDelegate.navigatorKey.currentContext!;
      showGeneralDialog<void>(
        context: context,
        pageBuilder: (_, _, _) => const Center(child: Text('Dialog content')),
      );
      await tester.pumpAndSettle();
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(find.text('Dialog content'), findsNothing);
      expect(find.byType(AppToastContent), findsNothing);
      expect(exits, isEmpty);
    },
    variant: TargetPlatformVariant.only(TargetPlatform.android),
  );

  testWidgets('bottom prompt stays above the Android safe area', (tester) async {
    tester.view.padding = const FakeViewPadding(bottom: 48);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPadding);
    addTearDown(tester.view.resetDevicePixelRatio);
    await mount(tester);
    await tester.binding.handlePopRoute();
    await tester.pump();
    expect(tester.getBottomLeft(find.byType(AppToastContent)).dy, tester.view.physicalSize.height - 48 - 24);
    await tester.pumpWidget(const SizedBox());
  }, variant: TargetPlatformVariant.only(TargetPlatform.android));
}
