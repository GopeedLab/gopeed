import 'dart:io';

import 'package:flutter/material.dart' show MaterialApp, Tooltip;
import 'package:flutter/widgets.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/widgets/app_tooltip.dart';

void main() {
  testWidgets('AppTooltip delegates product tooltips to Flutter Tooltip', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: AppTooltip(message: 'Details', child: SizedBox.square(dimension: 24)),
      ),
    );

    expect(tester.widget<Tooltip>(find.byType(Tooltip)).message, 'Details');
  });

  testWidgets('focusable status tooltip exposes its message on keyboard focus', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: AppTooltip(message: 'Waiting for resolution', focusable: true, child: SizedBox.square(dimension: 24)),
      ),
    );
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pumpAndSettle();
    expect(find.text('Waiting for resolution'), findsOneWidget);
    FocusManager.instance.primaryFocus?.unfocus();
    await tester.pumpAndSettle();
    expect(find.text('Waiting for resolution'), findsNothing);
  });

  testWidgets('status tooltip can be read with a touch long press', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Center(
          child: AppTooltip(message: 'Resolution failed', focusable: true, child: SizedBox.square(dimension: 24)),
        ),
      ),
    );
    await tester.longPress(find.byType(AppTooltip));
    await tester.pump();
    expect(find.text('Resolution failed'), findsOneWidget);
    Tooltip.dismissAllToolTips();
    await tester.pumpAndSettle();
  });

  test('product source does not instantiate Tooltip directly', () {
    final violations = <String>[];
    final tooltipConstructor = RegExp(r'\b(?:\w+\.)?Tooltip\s*\(');

    for (final entity in Directory('lib').listSync(recursive: true)) {
      if (entity is! File || !entity.path.endsWith('.dart') || entity.path.endsWith('app_tooltip.dart')) {
        continue;
      }
      final lines = entity.readAsLinesSync();
      for (var index = 0; index < lines.length; index++) {
        if (tooltipConstructor.hasMatch(lines[index])) {
          violations.add('${entity.path}:${index + 1}');
        }
      }
    }

    expect(
      violations,
      isEmpty,
      reason:
          'Use AppTooltip so product tooltip styling stays centralized. Direct Tooltip usage found at: '
          '${violations.join(', ')}',
    );
  });
}
