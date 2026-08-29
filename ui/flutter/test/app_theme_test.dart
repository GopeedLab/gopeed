import 'package:flutter/foundation.dart' show TargetPlatform, debugDefaultTargetPlatformOverride;
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_palette.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/detail/app_detail_surface_desktop.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  test('Android themes use platform fonts', () {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    try {
      final typography = AppTheme.light().typography;

      expect(typography.sans.fontFamily, isNull);
      expect(typography.mono.fontFamily, 'monospace');
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });

  test('Windows themes use a deterministic CJK font fallback', () {
    debugDefaultTargetPlatformOverride = TargetPlatform.windows;
    try {
      final shadTheme = AppTheme.light();
      final materialTheme = AppTheme.materialLight();

      expect(shadTheme.typography.sans.fontFamily, 'packages/shadcn_flutter/GeistSans');
      expect(shadTheme.typography.sans.fontFamilyFallback, const ['Microsoft YaHei UI']);
      expect(materialTheme.textTheme.bodyMedium?.fontFamilyFallback, const ['Microsoft YaHei UI']);
      expect(materialTheme.primaryTextTheme.bodyMedium?.fontFamilyFallback, const ['Microsoft YaHei UI']);
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });

  test('light theme separates navigation, canvas, and raised surfaces', () {
    const palette = AppPalette.light;

    expect(palette.railBg, isNot(palette.sideBg));
    expect(palette.sideBg, isNot(palette.bg));
    expect(palette.bg, isNot(palette.cardBg));
    expect(palette.railBg, isNot(palette.cardBg));
  });

  testWidgets('desktop detail drawer is raised only in the light theme', (tester) async {
    const drawerKey = ValueKey('palette-test-detail-drawer');

    Future<BoxDecoration> pumpDrawer({required bool light}) async {
      final palette = light ? AppPalette.light : AppPalette.dark;
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: light ? AppTheme.light() : AppTheme.dark(),
          materialTheme: light ? AppTheme.materialLight() : AppTheme.materialDark(),
          home: SizedBox.expand(
            child: AppDetailDrawer(
              open: true,
              title: 'Details',
              onClose: () {},
              drawerKey: drawerKey,
              child: const SizedBox(),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final drawer = tester.widget<Container>(find.byKey(drawerKey));
      final decoration = drawer.decoration! as BoxDecoration;
      final border = decoration.border! as Border;
      expect(border.left.color, light ? palette.headerDivider : palette.border);
      expect(border.top.color, palette.headerDivider);
      expect(find.byKey(const ValueKey('app-detail-drawer-close-button')), findsOneWidget);
      return decoration;
    }

    final lightDecoration = await pumpDrawer(light: true);
    expect(lightDecoration.color, AppPalette.light.cardBg);
    expect(lightDecoration.boxShadow, isNotEmpty);

    final darkDecoration = await pumpDrawer(light: false);
    expect(darkDecoration.color, AppPalette.dark.bg);
    expect(darkDecoration.boxShadow, isNull);
  });
}
