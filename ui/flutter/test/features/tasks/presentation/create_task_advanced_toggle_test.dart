import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/presentation/widgets/create_task_advanced_toggle.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final width in [360.0, 768.0, 1024.0]) {
    testWidgets('advanced disclosure supports keyboard and touch at width $width', (tester) async {
      tester.view.physicalSize = Size(width, 768);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      var expanded = false;
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(
            child: StatefulBuilder(
              builder: (context, setState) => Center(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: CreateTaskAdvancedToggle(
                    expanded: expanded,
                    onPressed: () => setState(() => expanded = !expanded),
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.getSize(find.byType(shad.GhostButton)).height, greaterThanOrEqualTo(44));
      expect(
        find.byWidgetPredicate((widget) => widget is Semantics && widget.properties.expanded == false),
        findsOneWidget,
      );
      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.pump();
      await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      await tester.pumpAndSettle();
      expect(expanded, isTrue);
      expect(
        find.byWidgetPredicate((widget) => widget is Semantics && widget.properties.expanded == true),
        findsOneWidget,
      );
      await tester.tap(find.text('Advanced'));
      await tester.pumpAndSettle();
      expect(expanded, isFalse);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    });
  }
}
