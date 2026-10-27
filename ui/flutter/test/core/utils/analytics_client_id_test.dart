import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/core/utils/analytics_client_id.dart';

void main() {
  test('different Android installations with the same build get different IDs', () {
    const sharedBuildId = 'AP3A.241105.008';

    final firstId = resolveAndroidAnalyticsClientId(sharedBuildId);
    final secondId = resolveAndroidAnalyticsClientId(sharedBuildId);

    expect(firstId, isNot(sharedBuildId));
    expect(secondId, isNot(firstId));
    expect(firstId, matches(r'^android-v1-[0-9a-f]{32}$'));
  });

  test('first launch generates an installation ID', () {
    expect(resolveAndroidAnalyticsClientId(''), matches(r'^android-v1-[0-9a-f]{32}$'));
  });

  test('migration reuses the existing config field and survives reloads', () {
    final config = DownloaderConfig()..extra.analyticsClientId = 'AP3A.241105.008';
    final installationId = resolveAndroidAnalyticsClientId(config.extra.analyticsClientId);
    config.extra.analyticsClientId = installationId;

    final reloaded = DownloaderConfig.fromJson(config.toJson());

    expect(resolveAndroidAnalyticsClientId(reloaded.extra.analyticsClientId), installationId);
  });

  test('migrates legacy fallback IDs and malformed installation IDs once', () {
    for (final legacyId in ['123456789.1700000000', 'android-v1-', 'android-v1-invalid']) {
      final installationId = resolveAndroidAnalyticsClientId(legacyId);

      expect(installationId, isNot(legacyId));
      expect(resolveAndroidAnalyticsClientId(installationId), installationId);
    }
  });
}
