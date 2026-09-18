import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/presentation/widgets/history_search_field.dart';
import 'package:gopeed/l10n/l10n.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final platform in [TargetPlatform.android, TargetPlatform.iOS, TargetPlatform.macOS]) {
    for (final scale in [1.0, 1.5, 2.0]) {
      testWidgets('history hint and input line align on $platform at scale $scale', (tester) async {
        debugDefaultTargetPlatformOverride = platform;
        addTearDown(() => debugDefaultTargetPlatformOverride = null);
        final controller = TextEditingController();
        addTearDown(controller.dispose);
        String? query;
        for (final width in [280.0, 360.0, 600.0]) {
          await tester.pumpWidget(
            shad.ShadcnApp(
              theme: AppTheme.light(),
              materialTheme: AppTheme.materialLight(),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: AppComponentThemes(
                child: Builder(
                  builder: (context) => Localizations.override(
                    context: context,
                    locale: const Locale('zh'),
                    child: MediaQuery(
                      data: MediaQueryData(textScaler: TextScaler.linear(scale)),
                      child: Center(
                        child: SizedBox(
                          width: width,
                          child: HistorySearchField(controller: controller, onChanged: (value) => query = value),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
          final editable = tester.state<EditableTextState>(find.byType(EditableText)).renderEditable;
          final hint = tester.renderObject<RenderBox>(find.text('搜索历史记录…'));
          expect(editable.preferredLineHeight, lessThanOrEqualTo(editable.size.height));
          expect(hint.size.height, closeTo(editable.preferredLineHeight, 0.1));
          expect(hint.localToGlobal(Offset.zero).dy, closeTo(editable.localToGlobal(Offset.zero).dy, 0.1));
          expect(tester.takeException(), isNull);
        }
        await tester.enterText(find.byType(EditableText), 'example');
        expect(query, 'example');
        debugDefaultTargetPlatformOverride = null;
      });
    }
  }
}
