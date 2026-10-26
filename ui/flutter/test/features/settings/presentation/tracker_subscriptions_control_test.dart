import 'package:flutter/gestures.dart' show PointerDeviceKind;
import 'package:flutter_test/flutter_test.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/features/settings/presentation/widgets/tracker_subscriptions_control.dart';
import 'package:gopeed/features/settings/presentation/widgets/tracker_subscriptions_schedule.dart';
import 'package:gopeed/features/settings/presentation/widgets/tracker_subscriptions_table.dart';
import 'package:gopeed/l10n/l10n.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';

void main() {
  for (final width in [360.0, 768.0, 1024.0]) {
    testWidgets('subscription results fit at width $width', (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final time = DateTime.utc(2026, 10, 19);
      await tester.pumpWidget(
        ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AppComponentThemes(
            child: Center(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: TrackerSubscriptionsControl(
                  urls: const [
                    'https://success.example/very/long/path/to/my/custom/list.txt',
                    'https://failed.example/list',
                    'https://new.example/list',
                  ],
                  statuses: {
                    'https://success.example/very/long/path/to/my/custom/list.txt': TrackerSubscriptionStatus(
                      success: true,
                      trackerCount: 7,
                      lastFetchTime: time,
                    ),
                    'https://failed.example/list': TrackerSubscriptionStatus(
                      success: false,
                      trackerCount: 0,
                      lastFetchTime: time,
                    ),
                  },
                  autoUpdate: true,
                  lastUpdated: time,
                  onChanged: (_) {},
                  onAutoUpdateChanged: (_) {},
                  onUpdate: () async {},
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Update status'), findsOneWidget);
      expect(find.text('Tracker count'), findsOneWidget);
      expect(find.text('Actions'), findsOneWidget);
      for (final entry in const [
        ('https://success.example/very/long/path/to/my/custom/list.txt', 'Succeeded', '7'),
        ('https://failed.example/list', 'Failed', '0'),
        ('https://new.example/list', 'Not updated', '—'),
      ]) {
        final row = find.byKey(ValueKey('tracker-subscription-${entry.$1}'));
        expect(find.descendant(of: row, matching: find.text(entry.$2)), findsOneWidget);
        expect(find.descendant(of: row, matching: find.text(entry.$3)), findsOneWidget);
      }
      final table = find.byKey(const ValueKey('tracker-subscriptions-table'));
      expect(find.descendant(of: table, matching: find.textContaining('2026-')), findsNothing);
      expect(find.textContaining('Last updated: '), findsOneWidget);
      final updateRect = tester.getRect(find.byKey(const ValueKey('tracker-update-button')));
      final addRect = tester.getRect(find.byKey(const ValueKey('add-tracker-subscription')));
      final tableRect = tester.getRect(table);
      final autoUpdateRect = tester.getRect(find.byKey(const ValueKey('tracker-auto-update-control')));
      final timeRect = tester.getRect(find.byKey(const ValueKey('tracker-last-updated')));
      expect(addRect.right, closeTo(tableRect.right, 1));
      expect(updateRect.right, lessThan(addRect.left));
      expect(addRect.center.dy, closeTo(updateRect.center.dy, 1));
      expect(updateRect.top, greaterThan(tableRect.bottom));
      expect(autoUpdateRect.bottom, lessThan(tableRect.top));
      expect(autoUpdateRect.right, closeTo(tableRect.right, 1));
      expect(timeRect.top, greaterThan(autoUpdateRect.bottom));
      expect(timeRect.bottom, lessThan(tableRect.top));
      expect(timeRect.right, closeTo(tableRect.right, 1));
      expect(find.byKey(const ValueKey('add-tracker-subscription')), findsOneWidget);
      if (width < 672) {
        final headerPosition = tester.getTopLeft(find.text('Subscription URL'));
        await tester.dragFrom(tester.getTopLeft(table) + const Offset(100, 20), const Offset(-600, 0));
        await tester.pumpAndSettle();
        expect(tester.getTopLeft(find.text('Subscription URL')).dx, lessThan(headerPosition.dx));
      }
      expect(tester.takeException(), isNull);
    });
  }
  for (final locale in const [Locale('en'), Locale('zh')]) {
    for (final width in [320.0, 768.0, 1024.0]) {
      testWidgets('schedule hides missing update time at width $width in $locale', (tester) async {
        tester.view.physicalSize = Size(width, 600);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        bool? autoUpdate;
        await tester.pumpWidget(
          ShadcnApp(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Builder(
              builder: (context) => Localizations.override(
                context: context,
                locale: locale,
                delegates: const [AppLocalizations.delegate],
                child: AppComponentThemes(
                  child: Center(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: SizedBox(
                        width: double.infinity,
                        child: TrackerSubscriptionsSchedule(
                          lastUpdated: null,
                          autoUpdate: false,
                          onAutoUpdateChanged: (value) => autoUpdate = value,
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(find.byKey(const ValueKey('tracker-last-updated')), findsNothing);
        expect(find.text('Never updated'), findsNothing);
        expect(find.text('尚未更新'), findsNothing);
        final scheduleRect = tester.getRect(find.byKey(const ValueKey('tracker-subscriptions-schedule')));
        final autoRect = tester.getRect(find.byKey(const ValueKey('tracker-auto-update-control')));
        expect(find.byKey(const ValueKey('tracker-update-button')), findsNothing);
        expect(autoRect.right, closeTo(scheduleRect.right, 1));
        await tester.tap(find.byType(Switch));
        expect(autoUpdate, true);
        expect(tester.takeException(), isNull);
      });
    }
  }
  for (final width in [360.0, 768.0, 1024.0]) {
    testWidgets('long subscription table scrolls within its viewport at width $width', (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final urls = List.generate(50, (index) => 'https://tracker-$index.example/list.txt');
      final deletions = <int>[];
      await tester.pumpWidget(
        ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AppComponentThemes(
            child: Center(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: TrackerSubscriptionsTable(urls: urls, statuses: const {}, onDelete: deletions.add),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final table = find.byKey(const ValueKey('tracker-subscriptions-table'));
      final rows = find.byKey(const ValueKey('tracker-subscriptions-rows'));
      expect(tester.getSize(table).height, lessThan(300));
      expect(tester.getSize(table).width, width - 32);
      final headerPosition = tester.getTopLeft(find.text('Subscription URL'));
      expect(find.byKey(ValueKey('tracker-subscription-${urls.last}')), findsNothing);
      await tester.dragFrom(tester.getTopLeft(rows) + const Offset(40, 80), const Offset(0, -3000));
      await tester.pumpAndSettle();
      expect(find.byKey(ValueKey('tracker-subscription-${urls.last}')), findsOneWidget);
      expect(tester.getTopLeft(find.text('Subscription URL')), headerPosition);
      final delete = find.byKey(ValueKey('delete-settings-list-${urls.last}'));
      final deleteRect = tester.getRect(delete);
      final actionsHeader = tester.getTopLeft(find.byKey(const ValueKey('tracker-subscriptions-actions-header')));
      expect(deleteRect.right, lessThanOrEqualTo(tester.getRect(table).right));
      expect(deleteRect.left, greaterThan(tester.getRect(table).left));
      expect(delete.hitTestable(), findsOneWidget);
      // Data columns scroll, while the delete column remains visible and fixed.
      if (width < 672) {
        final headerPosition = tester.getTopLeft(find.text('Subscription URL'));
        await tester.dragFrom(tester.getTopLeft(table) + const Offset(100, 20), const Offset(-30, 0));
        await tester.pumpAndSettle();
        await tester.dragFrom(
          tester.getBottomLeft(table) + const Offset(40, -5),
          const Offset(600, 0),
          kind: PointerDeviceKind.mouse,
        );
        await tester.pumpAndSettle();
        expect(tester.getTopLeft(find.text('Subscription URL')).dx, lessThan(headerPosition.dx));
      }
      expect(tester.getRect(delete), deleteRect);
      expect(tester.getTopLeft(find.byKey(const ValueKey('tracker-subscriptions-actions-header'))), actionsHeader);
      expect(find.byIcon(Icons.edit_outlined), findsNothing);
      if (width < 672) {
        final rightInset = tester.getRect(table).right - deleteRect.right;
        tester.view.physicalSize = const Size(768, 900);
        await tester.pumpAndSettle();
        expect(delete.hitTestable(), findsOneWidget);
        expect(tester.getRect(table).right - tester.getRect(delete).right, closeTo(rightInset, 1));
      }
      await tester.tap(delete);
      expect(deletions, [49]);
      expect(tester.takeException(), isNull);
    });
  }
}
