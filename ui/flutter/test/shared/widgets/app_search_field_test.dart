import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart' as material show AdaptiveTextSelectionToolbar, Icons;
import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_search_field.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final direction in TextDirection.values) {
    for (final width in [240.0, 480.0]) {
      testWidgets(
        'scrolling and selecting long search text preserves the icon ($direction, $width)',
        (tester) async {
          final controller = TextEditingController();
          final focus = FocusNode();
          addTearDown(controller.dispose);
          addTearDown(focus.dispose);
          await _pumpField(tester, controller, focus, width: width, direction: direction);
          focus.requestFocus();
          await tester.pumpAndSettle();
          final iconBefore = await _iconPixels(tester);
          final iconRect = tester.getRect(find.byIcon(material.Icons.search_rounded));
          final text = List.filled(
            20,
            direction == TextDirection.ltr ? 'long-search-text-' : 'نص البحث الطويل ',
          ).join();
          controller.value = TextEditingValue(
            text: text,
            selection: TextSelection.collapsed(offset: text.length),
          );
          await tester.pumpAndSettle();
          final editable = tester.state<EditableTextState>(find.byType(EditableText));
          expect(editable.renderEditable.maxScrollExtent, greaterThan(0));
          editable.renderEditable.offset.jumpTo(editable.renderEditable.maxScrollExtent);
          await tester.pumpAndSettle();
          expect(editable.renderEditable.offset.pixels, greaterThan(0));
          expect(tester.getRect(find.byIcon(material.Icons.search_rounded)), iconRect);
          expect(await _iconPixels(tester), orderedEquals(iconBefore));
          controller.selection = TextSelection(baseOffset: 0, extentOffset: text.length);
          await tester.pumpAndSettle();
          expect(await _iconPixels(tester), orderedEquals(iconBefore));
          expect(tester.takeException(), isNull);
        },
        variant: TargetPlatformVariant({TargetPlatform.android, TargetPlatform.macOS}),
      );
    }
  }

  testWidgets(
    'search text keeps Android native selection actions',
    (tester) async {
      final controller = TextEditingController(text: 'search selection');
      final focus = FocusNode();
      addTearDown(controller.dispose);
      addTearDown(focus.dispose);
      await _pumpField(tester, controller, focus);
      await tester.longPress(find.byType(EditableText));
      await tester.pumpAndSettle();
      expect(find.byType(material.AdaptiveTextSelectionToolbar), findsOneWidget);
      expect(find.byType(shad.MobileEditableTextContextMenu), findsNothing);
    },
    skip: kIsWeb,
    variant: TargetPlatformVariant({TargetPlatform.android}),
  );

  testWidgets(
    'search text keeps desktop native selection actions',
    (tester) async {
      final controller = TextEditingController(text: 'search selection');
      final focus = FocusNode();
      addTearDown(controller.dispose);
      addTearDown(focus.dispose);
      await _pumpField(tester, controller, focus);
      final gesture = await tester.startGesture(
        tester.getCenter(find.byType(EditableText)),
        kind: PointerDeviceKind.mouse,
        buttons: kSecondaryMouseButton,
      );
      await gesture.up();
      await tester.pumpAndSettle();
      expect(find.byType(material.AdaptiveTextSelectionToolbar), findsOneWidget);
      expect(find.byType(shad.DesktopEditableTextContextMenu), findsNothing);
    },
    skip: kIsWeb,
    variant: TargetPlatformVariant({TargetPlatform.macOS}),
  );
}

Future<void> _pumpField(
  WidgetTester tester,
  TextEditingController controller,
  FocusNode focus, {
  double width = 320,
  TextDirection direction = TextDirection.ltr,
}) async {
  await tester.pumpWidget(
    shad.ShadcnApp(
      theme: AppTheme.light(),
      materialTheme: AppTheme.materialLight(),
      home: AppComponentThemes(
        child: Directionality(
          textDirection: direction,
          child: Center(
            child: RepaintBoundary(
              key: const ValueKey('search-paint-boundary'),
              child: SizedBox(
                width: width,
                child: AppSearchField(controller: controller, focusNode: focus, placeholder: const Text('Search')),
              ),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<List<int>> _iconPixels(WidgetTester tester) async {
  final boundaryFinder = find.byKey(const ValueKey('search-paint-boundary'));
  final boundary = tester.renderObject<RenderRepaintBoundary>(boundaryFinder);
  final rect = tester.getRect(find.byIcon(material.Icons.search_rounded)).shift(-tester.getTopLeft(boundaryFinder));
  final snapshot = await tester.runAsync(() async {
    final image = await boundary.toImage(pixelRatio: 1);
    final width = image.width;
    final bytes = (await image.toByteData(format: ui.ImageByteFormat.rawRgba))!.buffer.asUint8List();
    image.dispose();
    return (width: width, pixels: Uint8List.fromList(bytes));
  });
  return [
    for (var y = rect.top.floor(); y < rect.bottom.ceil(); y++)
      for (var x = rect.left.floor(); x < rect.right.ceil(); x++)
        ...snapshot!.pixels.sublist((y * snapshot.width + x) * 4, (y * snapshot.width + x) * 4 + 4),
  ];
}
