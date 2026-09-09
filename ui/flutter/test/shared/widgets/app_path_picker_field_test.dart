import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_path_picker_field.dart';
import 'package:gopeed/shared/widgets/app_text_field.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('file paths remain editable and Web hides native file selection', (tester) async {
    final controller = TextEditingController(text: '/scripts/old.sh');
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: AppComponentThemes(
          child: Center(
            child: AppPathPickerField.file(
              controller: controller,
              desktopWidth: 320,
              pickerKey: const ValueKey('file-picker'),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byKey(const ValueKey('file-picker')), kIsWeb ? findsNothing : findsOneWidget);
    expect(tester.widget<AppTextField>(find.byType(AppTextField)).readOnly, isFalse);
    await tester.enterText(find.byType(EditableText), '/scripts/custom.sh');
    await tester.pump();
    expect(controller.text, '/scripts/custom.sh');
    expect(tester.takeException(), isNull);
  });
}
