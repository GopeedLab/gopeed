import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_design_tokens.dart';
import 'package:gopeed/shared/theme/app_palette.dart';
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

  testWidgets('uses a high-contrast checkbox border in the dark theme', (tester) async {
    late shad.CheckboxTheme checkboxTheme;

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.dark(),
        materialTheme: AppTheme.materialDark(),
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

    expect(checkboxTheme.borderColor, AppPalette.dark.textMuted);
    expect(_contrastRatio(checkboxTheme.borderColor!, AppPalette.dark.bg), greaterThanOrEqualTo(3));
  });
}

double _contrastRatio(Color first, Color second) {
  final firstLuminance = first.computeLuminance();
  final secondLuminance = second.computeLuminance();
  final lighter = firstLuminance > secondLuminance ? firstLuminance : secondLuminance;
  final darker = firstLuminance > secondLuminance ? secondLuminance : firstLuminance;
  return (lighter + 0.05) / (darker + 0.05);
}
