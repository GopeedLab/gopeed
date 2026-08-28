import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_palette.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_loading_button.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('loading buttons replace only the icon and preserve their color family', (tester) async {
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.dark(),
        materialTheme: AppTheme.materialDark(),
        home: AppComponentThemes(
          child: Column(
            children: [
              AppLoadingButton(
                key: const ValueKey('secondary-loading-button'),
                onPressed: () {},
                loading: true,
                icon: const Icon(shad.LucideIcons.refreshCw),
                child: const Text('Check for Updates'),
              ),
              AppLoadingButton(
                key: const ValueKey('brand-loading-button'),
                onPressed: () {},
                loading: true,
                variant: AppLoadingButtonVariant.brand,
                icon: const Icon(shad.LucideIcons.download),
                child: const Text('Update'),
              ),
              AppLoadingButton(
                key: const ValueKey('primary-loading-button'),
                onPressed: () {},
                loading: true,
                variant: AppLoadingButtonVariant.primary,
                icon: const Icon(shad.LucideIcons.save),
                child: const Text('Save'),
              ),
            ],
          ),
        ),
      ),
    );
    await tester.pump();

    final secondary = find.byKey(const ValueKey('secondary-loading-button'));
    final brand = find.byKey(const ValueKey('brand-loading-button'));
    final primary = find.byKey(const ValueKey('primary-loading-button'));
    expect(find.descendant(of: secondary, matching: find.text('Check for Updates')), findsOneWidget);
    expect(find.descendant(of: brand, matching: find.text('Update')), findsOneWidget);
    expect(find.descendant(of: primary, matching: find.text('Save')), findsOneWidget);
    expect(find.descendant(of: secondary, matching: find.byIcon(shad.LucideIcons.refreshCw)), findsNothing);
    expect(find.descendant(of: brand, matching: find.byIcon(shad.LucideIcons.download)), findsNothing);
    expect(find.descendant(of: primary, matching: find.byIcon(shad.LucideIcons.save)), findsNothing);

    final secondaryProgress = tester.widget<shad.CircularProgressIndicator>(
      find.descendant(of: secondary, matching: find.byType(shad.CircularProgressIndicator)),
    );
    final brandProgress = tester.widget<shad.CircularProgressIndicator>(
      find.descendant(of: brand, matching: find.byType(shad.CircularProgressIndicator)),
    );
    final primaryProgress = tester.widget<shad.CircularProgressIndicator>(
      find.descendant(of: primary, matching: find.byType(shad.CircularProgressIndicator)),
    );
    expect(secondaryProgress.color, AppPalette.dark.textMuted);
    expect(secondaryProgress.color, isNot(AppPalette.dark.brand));
    expect(brandProgress.color, AppPalette.dark.brandForeground);
    expect(primaryProgress.color, AppPalette.dark.primaryActionForeground);

    final secondaryDecoration =
        tester
                .widget<shad.Clickable>(find.descendant(of: secondary, matching: find.byType(shad.Clickable)))
                .decoration!
                .resolve({WidgetState.disabled})
            as BoxDecoration;
    final brandDecoration =
        tester
                .widget<shad.Clickable>(find.descendant(of: brand, matching: find.byType(shad.Clickable)))
                .decoration!
                .resolve({WidgetState.disabled})
            as BoxDecoration;
    expect(secondaryDecoration.color, AppPalette.dark.surfaceSoft);
    expect(brandDecoration.color, AppPalette.dark.brand);
  });
}
