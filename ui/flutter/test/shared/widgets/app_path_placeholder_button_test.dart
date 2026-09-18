import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_path_placeholder_button.dart';

void main() {
  for (final width in [390.0, 1024.0]) {
    testWidgets('path placeholders explain and insert variables at width $width', (tester) async {
      tester.view.physicalSize = Size(width, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final controller = TextEditingController(text: '/downloads/replace');
      addTearDown(controller.dispose);
      controller.selection = const TextSelection(baseOffset: 11, extentOffset: 18);
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Center(child: AppPathPlaceholderButton(controller: controller)),
        ),
      );
      await tester.tap(find.text('Insert Placeholder'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(find.text('%year% — Current year'), findsOneWidget);
      expect(find.text('%month% — Current month (01-12)'), findsOneWidget);
      expect(find.text('%day% — Current day (01-31)'), findsOneWidget);
      expect(find.text('%date% — Full date (YYYY-MM-DD)'), findsOneWidget);
      expect(find.text('e.g. ${DateTime.now().year}'), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey('path-placeholder-%year%')));
      await tester.pumpAndSettle();
      expect(controller.text, '/downloads/%year%');
      expect(controller.selection.baseOffset, controller.text.length);
      expect(find.text('%year% — Current year'), findsNothing);
      for (final variable in ['%month%', '%day%', '%date%']) {
        controller.text = '/downloads/';
        await tester.tap(find.text('Insert Placeholder'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));
        await tester.tap(find.byKey(ValueKey('path-placeholder-$variable')));
        await tester.pumpAndSettle();
        expect(controller.text, '/downloads/$variable');
      }
      expect(tester.takeException(), isNull);
    });
  }
}
