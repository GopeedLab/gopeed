import 'package:flutter_test/flutter_test.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/features/settings/presentation/widgets/tracker_subscriptions_control.dart';
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
      expect(find.textContaining('Last fetch succeeded: 7 trackers'), findsOneWidget);
      expect(find.textContaining('Last fetch failed: 0 trackers'), findsOneWidget);
      expect(find.text('Never updated'), findsOneWidget);
      expect(find.byKey(const ValueKey('add-tracker-subscription')), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
}
