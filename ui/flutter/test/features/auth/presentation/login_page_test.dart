import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/auth/presentation/pages/login_page.dart';
import 'package:gopeed/l10n/app_localizations.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_loading_button.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  Widget buildApp() {
    return ProviderScope(
      child: shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const AppComponentThemes(child: LoginPage()),
      ),
    );
  }

  Future<void> setViewport(WidgetTester tester, Size size) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });
    await tester.pumpWidget(buildApp());
    await tester.pumpAndSettle();
  }

  testWidgets('uses a compact responsive card on mobile', (tester) async {
    await setViewport(tester, const Size(390, 844));

    expect(find.byKey(const ValueKey('login-compact-card')), findsOneWidget);
    expect(find.byKey(const ValueKey('login-desktop-card')), findsNothing);
    expect(find.byType(SingleChildScrollView), findsNothing);
    expect(find.byType(FittedBox), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('uses a split card on desktop', (tester) async {
    await setViewport(tester, const Size(1200, 800));

    expect(find.byKey(const ValueKey('login-desktop-card')), findsOneWidget);
    expect(find.byKey(const ValueKey('login-compact-card')), findsNothing);
    expect(find.byType(SingleChildScrollView), findsNothing);
    expect(find.byType(FittedBox), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('fits a short narrow viewport without overflow or scrolling', (tester) async {
    await setViewport(tester, const Size(320, 480));

    expect(find.byType(SingleChildScrollView), findsNothing);
    expect(find.byKey(const ValueKey('login-compact-card')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('exposes browser-compatible username and password autofill fields', (tester) async {
    await setViewport(tester, const Size(390, 844));

    final username = tester.widget<shad.TextField>(find.byKey(const ValueKey('login-username-field')));
    final password = tester.widget<shad.TextField>(find.byKey(const ValueKey('login-password-field')));

    final autofillGroup = tester.widget<AutofillGroup>(find.byType(AutofillGroup));
    expect(autofillGroup.onDisposeAction, AutofillContextAction.cancel);
    expect(username.autofillHints, contains(AutofillHints.username));
    expect(username.autofocus, isTrue);
    expect(username.textInputAction, TextInputAction.done);
    expect(password.autofillHints, contains(AutofillHints.password));
    expect(password.textInputAction, TextInputAction.done);
    expect(password.obscureText, isTrue);
    expect(password.features.whereType<shad.InputPasswordToggleFeature>(), hasLength(1));
  });

  testWidgets('uses a compact centered neutral login action', (tester) async {
    await setViewport(tester, const Size(1200, 800));

    final buttonContainer = find.byKey(const ValueKey('login-submit-button-container'));
    final button = tester.widget<AppLoadingButton>(find.byKey(const ValueKey('login-submit-button')));
    expect(tester.getSize(buttonContainer), const Size(148, 40));
    expect(button.alignment, Alignment.center);
    expect(button.variant, AppLoadingButtonVariant.primary);
  });

  testWidgets('validates required credentials before submitting', (tester) async {
    await setViewport(tester, const Size(390, 844));

    await tester.tap(find.byKey(const ValueKey('login-submit-button')));
    await tester.pump();

    expect(find.text('Please enter your username'), findsOneWidget);
    expect(find.text('Please enter your password'), findsOneWidget);
  });
}
