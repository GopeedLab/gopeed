import 'package:flutter_test/flutter_test.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/features/settings/presentation/widgets/tracker_subscriptions_control.dart';
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
      for (final entry in const [
        ('https://success.example/very/long/path/to/my/custom/list.txt', 'Succeeded', '7'),
        ('https://failed.example/list', 'Failed', '0'),
        ('https://new.example/list', 'Not fetched', '—'),
      ]) {
        final row = find.byKey(ValueKey('tracker-subscription-${entry.$1}'));
        expect(find.descendant(of: row, matching: find.text(entry.$2)), findsOneWidget);
        expect(find.descendant(of: row, matching: find.text(entry.$3)), findsOneWidget);
      }
      final table = find.byKey(const ValueKey('tracker-subscriptions-table'));
      expect(find.descendant(of: table, matching: find.textContaining('2026-')), findsNothing);
      expect(find.textContaining('Last updated: '), findsOneWidget);
      expect(find.byKey(const ValueKey('add-tracker-subscription')), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
  for (final width in [360.0, 768.0, 1024.0]) {
    testWidgets('long subscription table scrolls within its viewport at width $width', (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final urls = List.generate(50, (index) => 'https://tracker-$index.example/list.txt');
      final edits = <int>[];
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
                child: TrackerSubscriptionsTable(
                  urls: urls,
                  statuses: const {},
                  onEdit: edits.add,
                  onDelete: deletions.add,
                ),
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
      // Small screens can scroll across to the same actions as the desktop table.
      if (width < 672) {
        await tester.dragFrom(tester.getTopLeft(table) + const Offset(100, 20), const Offset(-600, 0));
        await tester.pumpAndSettle();
      }
      await tester.tap(find.byKey(ValueKey('edit-settings-list-${urls.last}')));
      await tester.tap(find.byKey(ValueKey('delete-settings-list-${urls.last}')));
      expect(edits, [49]);
      expect(deletions, [49]);
      expect(tester.takeException(), isNull);
    });
  }
}
