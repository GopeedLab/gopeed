import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart' as material show AdaptiveTextSelectionToolbar;
import 'package:flutter/services.dart' show BrowserContextMenu;
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_text_field.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('Android long press uses Flutter adaptive text selection toolbar', (tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    final controller = TextEditingController(text: 'https://example.com/download');
    addTearDown(controller.dispose);

    try {
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(
            child: Center(
              child: SizedBox(width: 320, child: AppTextField(controller: controller)),
            ),
          ),
        ),
      );

      await tester.longPress(find.byType(AppTextField));
      await tester.pumpAndSettle();

      expect(find.byType(material.AdaptiveTextSelectionToolbar), findsOneWidget);
      expect(find.byType(shad.MobileEditableTextContextMenu), findsNothing);
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  }, skip: kIsWeb);

  testWidgets('desktop right click uses Flutter adaptive text selection toolbar', (tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.macOS;
    final controller = TextEditingController(text: 'https://example.com/download');
    addTearDown(controller.dispose);

    try {
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(
            child: Center(
              child: SizedBox(width: 320, child: AppTextField(controller: controller)),
            ),
          ),
        ),
      );

      final field = find.byType(AppTextField);
      final gesture = await tester.startGesture(
        tester.getCenter(field),
        kind: PointerDeviceKind.mouse,
        buttons: kSecondaryMouseButton,
      );
      await gesture.up();
      await tester.pumpAndSettle();

      expect(find.byType(material.AdaptiveTextSelectionToolbar), findsOneWidget);
      expect(find.byType(shad.DesktopEditableTextContextMenu), findsNothing);
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  }, skip: kIsWeb);

  testWidgets('Web right click keeps the browser menu and creates no Flutter overlay', (tester) async {
    final controller = TextEditingController(text: 'https://example.com/download');
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      shad.ShadcnApp(
        disableBrowserContextMenu: false,
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: AppComponentThemes(
          child: Center(
            child: SizedBox(width: 320, child: AppTextField(controller: controller)),
          ),
        ),
      ),
    );

    await tester.pumpAndSettle();
    expect(BrowserContextMenu.enabled, isTrue);
    final gesture = await tester.startGesture(
      tester.getCenter(find.byType(AppTextField)),
      kind: PointerDeviceKind.mouse,
      buttons: kSecondaryMouseButton,
    );
    await gesture.up();
    await tester.pumpAndSettle();

    expect(find.byType(material.AdaptiveTextSelectionToolbar), findsNothing);
    expect(find.byType(shad.DesktopEditableTextContextMenu), findsNothing);
    expect(find.byType(shad.MobileEditableTextContextMenu), findsNothing);
  }, skip: !kIsWeb);

  test('AppTextField defaults to the shared context menu policy', () {
    expect(const AppTextField().contextMenuBuilder, same(appTextFieldContextMenuBuilder));
  });
}
