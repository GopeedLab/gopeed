import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_design_tokens.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('provides the standard checkbox metrics', (tester) async {
    late shad.CheckboxTheme checkboxTheme;

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: AppComponentThemes(
          child: Builder(
            builder: (context) {
              checkboxTheme = shad.ComponentTheme.of<shad.CheckboxTheme>(context);
              return const SizedBox();
            },
          ),
        ),
      ),
    );

    expect(checkboxTheme.size, AppDesignTokens.checkboxSize);
    expect(checkboxTheme.gap, AppDesignTokens.checkboxLabelGap);
  });
}
