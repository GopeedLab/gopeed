import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import 'package:gopeed/features/settings/presentation/widgets/extension_tag_editor.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';

Widget _wrap(Widget child) => shad.ShadcnApp(
  theme: AppTheme.light(),
  materialTheme: AppTheme.materialLight(),
  home: AppComponentThemes(child: child),
);

List<String> _chipTexts(WidgetTester tester) => tester
    .widgetList<Text>(find.descendant(of: find.byType(Wrap), matching: find.byType(Text)))
    .map((text) => text.data ?? '')
    .toList();

Color _chipBorder(WidgetTester tester, String tag) {
  final chip = tester.widget<Container>(find.byKey(ValueKey('extension-chip-$tag')));
  final border = (chip.decoration! as BoxDecoration).border! as Border;
  return border.top.color;
}

void main() {
  testWidgets('separators stay in the input until Enter commits them as tags', (WidgetTester tester) async {
    await tester.pumpWidget(
      _wrap(ExtensionTagEditor(inputKey: const ValueKey('field'), initialTags: const ['.ZIP', ' Exe', 'exe'])),
    );
    await tester.pumpAndSettle();
    expect(_chipTexts(tester), ['zip', 'exe']);

    await tester.enterText(find.byKey(const ValueKey('field')), 'mp3, flac');
    await tester.pump();
    expect(_chipTexts(tester), ['zip', 'exe']);
    expect(find.text('mp3, flac'), findsOneWidget);

    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();
    expect(_chipTexts(tester), ['zip', 'exe', 'mp3', 'flac']);
    expect(find.text('mp3, flac'), findsNothing);

    await tester.enterText(find.byKey(const ValueKey('field')), '.APE');
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();
    expect(_chipTexts(tester), ['zip', 'exe', 'mp3', 'flac', 'ape']);

    await tester.tap(find.byKey(const ValueKey('extension-chip-remove-exe')));
    await tester.pump();
    expect(_chipTexts(tester), ['zip', 'mp3', 'flac', 'ape']);
    expect(find.byKey(const ValueKey('extension-chip-remove-exe')), findsNothing);

    await tester.enterText(find.byKey(const ValueKey('field')), 'DO');
    final state = tester.state<ExtensionTagEditorState>(find.byType(ExtensionTagEditor));
    expect(state.flush(), ['zip', 'mp3', 'flac', 'ape', 'do']);
    await tester.pump();
    expect(_chipTexts(tester), ['zip', 'mp3', 'flac', 'ape', 'do']);
    expect(tester.takeException(), isNull);
  });

  testWidgets('typing a duplicate highlights the existing chip without adding it', (WidgetTester tester) async {
    await tester.pumpWidget(
      _wrap(ExtensionTagEditor(inputKey: const ValueKey('field'), initialTags: const ['zip', 'exe'])),
    );
    await tester.pumpAndSettle();
    final zipIdle = _chipBorder(tester, 'zip');

    await tester.enterText(find.byKey(const ValueKey('field')), ' EXE ');
    await tester.pump();
    expect(_chipBorder(tester, 'exe'), isNot(zipIdle));
    expect(_chipBorder(tester, 'zip'), zipIdle);

    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();
    expect(_chipTexts(tester), ['zip', 'exe']);
    expect(_chipBorder(tester, 'exe'), zipIdle);
    expect(tester.takeException(), isNull);
  });

  testWidgets('placeholder surfaces default extensions only while the editor is empty', (WidgetTester tester) async {
    await tester.pumpWidget(
      _wrap(
        ExtensionTagEditor(
          inputKey: const ValueKey('field'),
          initialTags: const [],
          placeholderText: 'empty hint',
          usagePlaceholderText: 'usage hint',
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('empty hint'), findsOneWidget);

    await tester.enterText(find.byKey(const ValueKey('field')), 'a');
    await tester.pump();
    expect(find.text('empty hint'), findsNothing);
    expect(find.text('usage hint'), findsNothing);
    expect(_chipTexts(tester), isEmpty);
    expect(find.byKey(const ValueKey('extension-restore-defaults')), findsNothing);

    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();
    expect(_chipTexts(tester), ['a']);
    expect(find.text('empty hint'), findsNothing);
    expect(find.text('usage hint'), findsOneWidget);

    await tester.enterText(find.byKey(const ValueKey('field')), 'b');
    await tester.pump();
    expect(find.text('usage hint'), findsNothing);

    await tester.enterText(find.byKey(const ValueKey('field')), '');
    await tester.pump();
    expect(find.text('usage hint'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('extension-chip-remove-a')));
    await tester.pump();
    expect(find.text('empty hint'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('empty state offers a restore-defaults action', (WidgetTester tester) async {
    await tester.pumpWidget(
      _wrap(
        ExtensionTagEditor(
          inputKey: const ValueKey('field'),
          initialTags: const [],
          placeholderText: 'empty state',
          restoreDefaults: const ['Exe', '.MSI', 'exe'],
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('empty state'), findsOneWidget);
    expect(find.byKey(const ValueKey('extension-restore-defaults')), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('extension-restore-defaults')));
    await tester.pump();
    expect(_chipTexts(tester), ['exe', 'msi']);
    expect(find.text('empty state'), findsNothing);
    expect(find.byKey(const ValueKey('extension-restore-defaults')), findsNothing);

    await tester.tap(find.byKey(const ValueKey('extension-chip-remove-exe')));
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('extension-chip-remove-msi')));
    await tester.pump();
    expect(find.text('empty state'), findsOneWidget);
    expect(find.byKey(const ValueKey('extension-restore-defaults')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
