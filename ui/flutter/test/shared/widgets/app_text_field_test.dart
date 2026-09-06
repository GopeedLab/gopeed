import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' as material show AdaptiveTextSelectionToolbar;
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_text_field.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  test('native context menus are limited to Android and iOS', () {
    expect(appTextFieldUsesNativeContextMenu(platform: TargetPlatform.android, isWeb: false), isTrue);
    expect(appTextFieldUsesNativeContextMenu(platform: TargetPlatform.iOS, isWeb: false), isTrue);
    expect(appTextFieldUsesNativeContextMenu(platform: TargetPlatform.macOS, isWeb: false), isFalse);
    expect(appTextFieldUsesNativeContextMenu(platform: TargetPlatform.windows, isWeb: false), isFalse);
    expect(appTextFieldUsesNativeContextMenu(platform: TargetPlatform.linux, isWeb: false), isFalse);
    expect(appTextFieldUsesNativeContextMenu(platform: TargetPlatform.android, isWeb: true), isFalse);
  });

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
  });

  test('AppTextField defaults to the shared context menu policy', () {
    expect(const AppTextField().contextMenuBuilder, same(appTextFieldContextMenuBuilder));
  });
}
