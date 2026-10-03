import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/app/application/downloader_config_init.dart';
import 'package:path/path.dart' as path;

void main() {
  DownloaderConfig configWith({List<DownloadCategory> categories = const [], String downloadDir = ''}) {
    final config = DownloaderConfig(downloadDir: downloadDir)..categories = List.of(categories);
    return config;
  }

  group('loadDownloaderConfig', () {
    test('persists when the load succeeded and init changed the config', () async {
      var saved = 0;
      final config = await loadDownloaderConfig(
        load: () async => configWith(),
        save: (_) async => saved++,
        defaultDownloadDir: () async => '/downloads',
      );
      expect(saved, 1);
      expect(config.categories, hasLength(4));
    });

    test('does not persist when the initial load failed', () async {
      var saved = 0;
      final config = await loadDownloaderConfig(
        load: () async => throw StateError('backend unavailable'),
        save: (_) async => saved++,
        defaultDownloadDir: () async => '/downloads',
      );
      expect(saved, 0);
      // The in-memory fallback still gets usable defaults.
      expect(config.downloadDir, '/downloads');
      expect(config.categories, hasLength(4));
    });

    test('skips persist when init changed nothing', () async {
      var saved = 0;
      await loadDownloaderConfig(
        load: () async => configWith(
          categories: [
            DownloadCategory(name: 'Mine', path: '/downloads/Mine', extensions: const ['zip']),
          ],
          downloadDir: '/downloads',
        ),
        save: (_) async => saved++,
        defaultDownloadDir: () async => '/other',
      );
      expect(saved, 0);
    });
  });

  group('initDownloaderConfig', () {
    test('seeds built-in categories with default extensions when empty', () async {
      final config = configWith();
      final changed = await initDownloaderConfig(config, defaultDownloadDir: () async => '/downloads');

      expect(changed, isTrue);
      expect(config.categories.map((c) => c.nameKey).toList(), [
        'categoryMusic',
        'categoryVideo',
        'categoryDocument',
        'categoryProgram',
      ]);
      for (final category in config.categories) {
        expect(category.isBuiltIn, isTrue);
        expect(category.path, path.join('/downloads', category.nameKey!.replaceFirst('category', '')));
        expect(category.extensions, kDefaultCategoryExtensions[category.nameKey]);
      }
    });

    test('keeps existing categories untouched', () async {
      final existing = DownloadCategory(name: 'Mine', path: '/downloads/Mine', extensions: const ['zip']);
      final config = configWith(categories: [existing], downloadDir: '/downloads');
      final changed = await initDownloaderConfig(config, defaultDownloadDir: () async => '/other');

      expect(changed, isFalse);
      expect(config.categories, hasLength(1));
      expect(config.categories.single, same(existing));
    });
  });
}
