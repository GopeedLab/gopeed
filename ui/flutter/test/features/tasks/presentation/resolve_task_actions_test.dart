import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/presentation/widgets/create_task_action_buttons.dart';
import 'package:gopeed/l10n/l10n.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_loading_button.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final width in [390.0, 1024.0]) {
    for (final language in ['en', 'zh', 'de']) {
      testWidgets('resolve actions stay equal at $width in $language, including loading', (tester) async {
        tester.view.physicalSize = Size(width, 760);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        var submitting = false;
        var cancelled = false;
        await tester.pumpWidget(
          shad.ShadcnApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            home: Builder(
              builder: (context) => Localizations.override(
                context: context,
                locale: Locale(language),
                delegates: const [AppLocalizations.delegate],
                child: AppComponentThemes(
                  child: Center(
                    child: StatefulBuilder(
                      builder: (context, setState) => shad.AlertDialog(
                        padding: const EdgeInsets.all(18),
                        content: SizedBox(width: width < 760 ? width - 32 : 720, height: 100),
                        actions: [
                          Flexible(
                            child: CreateTaskActionButtons(
                              submitting: submitting,
                              onCancel: () => cancelled = true,
                              onSubmit: () => setState(() => submitting = true),
                              cancelLabel: context.l10n.cancel,
                              submitLabel: context.l10n.createAction,
                              cancelButtonKey: const ValueKey('resolve-cancel-button'),
                              submitButtonKey: const ValueKey('resolve-create-button'),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final cancel = find.byKey(const ValueKey('resolve-cancel-button'));
        final create = find.byKey(const ValueKey('resolve-create-button'));
        final cancelRect = tester.getRect(cancel);
        final createRect = tester.getRect(create);
        expect(cancelRect.size, createRect.size);
        expect(cancelRect.center.dy, createRect.center.dy);
        expect(createRect.left - cancelRect.right, 12);
        expect((tester.widget<shad.SecondaryButton>(cancel).child as SizedBox).width, 68);
        expect((tester.widget<AppLoadingButton>(create).child as SizedBox).width, 68);
        expect(cancelRect.left, greaterThanOrEqualTo(0));
        expect(createRect.right, lessThanOrEqualTo(width));
        expect(find.text(appLocalizationsFor(language).createAction), findsOneWidget);
        await tester.tap(cancel);
        expect(cancelled, isTrue);
        await tester.tap(create);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 200));
        expect(tester.widget<AppLoadingButton>(create).loading, isTrue);
        expect(tester.widget<shad.SecondaryButton>(cancel).onPressed, isNull);
        expect(tester.getRect(cancel), cancelRect);
        expect(tester.getRect(create), createRect);
        expect(tester.takeException(), isNull);
      });
    }
  }
}
