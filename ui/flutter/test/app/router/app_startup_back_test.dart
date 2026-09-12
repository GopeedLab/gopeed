import 'dart:async';

import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/app/router/app_router.dart';
import 'package:gopeed/features/auth/application/web_auth_controller.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_toast.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

class _StartingRuntime extends AppRuntimeController {
  final pending = Completer<AppRuntimeState>();

  @override
  Future<AppRuntimeState> build() => pending.future;
}

void main() {
  testWidgets(
    'cold startup registers Android back and confirms exit before runtime is ready',
    (tester) async {
      final platformCalls = <MethodCall>[];
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(SystemChannels.platform, (call) async {
        platformCalls.add(call);
        return null;
      });
      addTearDown(() => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(SystemChannels.platform, null));
      final runtime = _StartingRuntime();
      final auth = WebAuthController();
      final router = AppRouter.build(auth);
      addTearDown(router.dispose);
      addTearDown(auth.dispose);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [appRuntimeControllerProvider.overrideWith(() => runtime)],
          child: shad.ShadcnApp.router(
            theme: AppTheme.dark(),
            materialTheme: AppTheme.materialDark(),
            routerConfig: router,
          ),
        ),
      );
      await tester.pump();
      await tester.pump();
      expect(find.byType(shad.CircularProgressIndicator), findsOneWidget);
      expect(
        platformCalls.where((call) => call.method == 'SystemNavigator.setFrameworkHandlesBack').last.arguments,
        isTrue,
      );
      await tester.binding.handlePopRoute();
      await tester.pump();
      expect(platformCalls.where((call) => call.method == 'SystemNavigator.pop'), isEmpty);
      expect(find.byType(AppToastContent), findsOneWidget);
      await tester.binding.handlePopRoute();
      await tester.pump();
      expect(platformCalls.where((call) => call.method == 'SystemNavigator.pop'), hasLength(1));
      await tester.pumpWidget(const SizedBox());
    },
    variant: TargetPlatformVariant.only(TargetPlatform.android),
  );
}
