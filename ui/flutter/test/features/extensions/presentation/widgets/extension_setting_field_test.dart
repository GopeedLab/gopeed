import 'package:flutter_test/flutter_test.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:gopeed/api/model/extension.dart';
import 'package:gopeed/features/extensions/presentation/widgets/extension_setting_field.dart';
import 'package:gopeed/shared/widgets/app_text_field.dart';

void main() {
  Future<void> pumpField(WidgetTester tester, Setting setting, TextEditingController controller) async {
    await tester.pumpWidget(
      ShadcnApp(
        home: Scaffold(
          child: Center(
            child: SizedBox(
              width: 280,
              child: ExtensionSettingField(setting: setting, controller: controller),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  Setting setting(SettingType type) =>
      Setting(name: 'test', title: 'Test setting', description: 'Description', required: false, type: type);

  testWidgets('boolean switch loads and updates the saved value', (tester) async {
    final controller = TextEditingController(text: 'true');
    addTearDown(controller.dispose);
    await pumpField(tester, setting(SettingType.boolean), controller);
    expect(find.byType(AppTextField), findsNothing);
    expect(tester.widget<Switch>(find.byType(Switch)).value, isTrue);
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();
    expect(controller.text, 'false');
    expect(tester.widget<Switch>(find.byType(Switch)).value, isFalse);
  });

  for (final type in [SettingType.string, SettingType.number]) {
    testWidgets(
      '$type options show labels and save selected values',
      (tester) async {
        final field = setting(type)..options = [Option(label: 'First', value: 1), Option(label: 'Second', value: 2)];
        final controller = TextEditingController(text: '1');
        addTearDown(controller.dispose);
        await pumpField(tester, field, controller);
        expect(find.byType(AppTextField), findsNothing);
        expect(find.text('First'), findsOneWidget);
        await tester.tap(find.byType(Select<String>));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 500));
        await tester.tap(find.text('Second'));
        await tester.pumpAndSettle();
        expect(controller.text, '2');
        expect(find.text('Second'), findsOneWidget);
        expect(tester.takeException(), isNull);
      },
      variant: TargetPlatformVariant({TargetPlatform.android, TargetPlatform.macOS}),
    );
  }

  testWidgets('number field accepts decimals and rejects nonnumeric input', (tester) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    await pumpField(tester, setting(SettingType.number), controller);
    await tester.enterText(find.byType(AppTextField), '-12.5');
    expect(controller.text, '-12.5');
    await tester.enterText(find.byType(AppTextField), 'invalid');
    expect(controller.text, '-12.5');
  });

  testWidgets('string with empty options remains a text field', (tester) async {
    final controller = TextEditingController(text: 'original');
    addTearDown(controller.dispose);
    await pumpField(tester, setting(SettingType.string)..options = [], controller);
    await tester.enterText(find.byType(AppTextField), 'updated');
    expect(controller.text, 'updated');
  });
}
