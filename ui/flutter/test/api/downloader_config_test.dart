import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:path/path.dart' as path;

void main() {
  test('auto-start tasks uses the backend-owned top-level config field', () {
    final config = DownloaderConfig(autoStartTasks: true);
    final json = config.toJson();

    expect(json['autoStartTasks'], isTrue);
    expect(json['extra'], isNot(contains('autoStartTasks')));
    expect(DownloaderConfig.fromJson(json).autoStartTasks, isTrue);
  });

  test('legacy extra auto-start setting is not migrated', () {
    final json = DownloaderConfig().toJson();
    (json['extra'] as Map<String, dynamic>)['autoStartTasks'] = true;

    expect(DownloaderConfig.fromJson(json).autoStartTasks, isFalse);
  });

  test('legacy extra downloadCategories is ignored on parse', () {
    final json = DownloaderConfig().toJson();
    (json['extra'] as Map<String, dynamic>)['downloadCategories'] = [
      {'name': 'Music', 'path': '/old/Music'},
    ];

    final decoded = DownloaderConfig.fromJson(json);
    expect(decoded.categories, isEmpty);
    expect(decoded.toJson()['extra'], isNot(contains('downloadCategories')));
  });

  test('REST server settings round-trip through the Go downloader config', () {
    final config = DownloaderConfig()
      ..api = ApiServerConfig(
        enable: true,
        mcpEnable: true,
        network: 'tcp',
        address: '127.0.0.1:4321',
        token: 'secret',
      );

    final decoded = DownloaderConfig.fromJson(config.toJson()).api;
    expect(decoded.enable, isTrue);
    expect(decoded.mcpEnable, isTrue);
    expect(decoded.network, 'tcp');
    expect(decoded.address, '127.0.0.1:4321');
    expect(decoded.token, 'secret');
  });

  test('Flutter preferences round-trip through the Go-owned extra config', () {
    final config = DownloaderConfig();
    config.extra
      ..windowState = WindowStateConfig(isMaximized: true, width: 1280, height: 720)
      ..bookmarks = {'downloads': 'D:/Downloads'}
      ..createHistory = ['https://example.com/file.zip']
      ..runAsMenubarApp = true
      ..analyticsEnabled = false
      ..analyticsClientId = 'client-id';

    final decoded = DownloaderConfig.fromJson(config.toJson()).extra;
    expect(decoded.windowState.isMaximized, isTrue);
    expect(decoded.windowState.width, 1280);
    expect(decoded.windowState.height, 720);
    expect(decoded.bookmarks, {'downloads': 'D:/Downloads'});
    expect(decoded.createHistory, ['https://example.com/file.zip']);
    expect(decoded.runAsMenubarApp, isTrue);
    expect(decoded.analyticsEnabled, isFalse);
    expect(decoded.analyticsClientId, 'client-id');
  });

  test('categories and the auto-categorize switch round-trip through the config', () {
    final config = DownloaderConfig()
      ..categories = [
        DownloadCategory(name: 'Program', path: '/dl/Program', extensions: const ['exe', 'msi']),
      ]
      ..autoCategorize = true;

    final decoded = DownloaderConfig.fromJson(config.toJson());
    expect(decoded.autoCategorize, isTrue);
    expect(decoded.categories.single.name, 'Program');
    expect(decoded.categories.single.extensions, ['exe', 'msi']);
  });

  test('config parses when the category fields are absent or null', () {
    final missing = DownloaderConfig().toJson()
      ..remove('categories')
      ..remove('autoCategorize');
    expect(DownloaderConfig.fromJson(missing).categories, isEmpty);
    expect(DownloaderConfig.fromJson(missing).autoCategorize, isTrue);

    // The Go backend marshals a nil category slice and an unset switch as null.
    final nil = DownloaderConfig().toJson()
      ..['categories'] = null
      ..['autoCategorize'] = null;
    expect(DownloaderConfig.fromJson(nil).categories, isEmpty);
    expect(DownloaderConfig.fromJson(nil).autoCategorize, isTrue);
  });

  test('auto categorize can be turned off and round-trips', () {
    final config = DownloaderConfig(autoCategorize: false);
    expect(DownloaderConfig.fromJson(config.toJson()).autoCategorize, isFalse);
  });

  test('constructor defaults auto categorize to true', () {
    expect(DownloaderConfig().autoCategorize, isTrue);
  });

  group('categoryCandidateFileName', () {
    test('prefers the rename when it carries an extension', () {
      expect(categoryCandidateFileName(rename: 'client.v2', url: 'https://example.com/setup.exe'), 'client.v2');
    });

    test('falls back to the url file name when the rename has no extension', () {
      expect(
        categoryCandidateFileName(rename: 'client', url: 'https://example.com/setup.exe?token=1#frag'),
        'setup.exe',
      );
      expect(categoryCandidateFileName(url: 'https://example.com/dir/final%20setup.exe'), 'final setup.exe');
    });

    test('returns empty for urls without a usable file name', () {
      expect(categoryCandidateFileName(url: 'magnet:?xt=urn:btih:abcdef'), '');
      expect(categoryCandidateFileName(url: 'data:application/octet-stream;base64,AAAA'), '');
      expect(categoryCandidateFileName(), '');
    });

    test('extracts the name from ed2k file shares', () {
      expect(categoryCandidateFileName(url: 'ed2k://|file|movie.avi|1024|0123456789ABCDEF|/'), 'movie.avi');
    });

    test('falls back to a file name carried in the query', () {
      expect(
        categoryCandidateFileName(
          url:
              'https://release-assets.githubusercontent.com/github-production-release-asset/212613049/'
              '8bf8c91c-fb1a-43ad-8eb7-3820af49e71d'
              '?response-content-disposition=attachment%3B%20filename%3Dgh_2.102.0_windows_amd64.msi',
        ),
        'gh_2.102.0_windows_amd64.msi',
      );
      expect(categoryCandidateFileName(url: 'https://example.com/token/8bf8c91c?filename=setup.exe'), 'setup.exe');
      expect(categoryCandidateFileName(url: 'https://example.com/token/8bf8c91c?filename=readme'), '');
    });
  });

  group('categoryFileNameFromQuery', () {
    test('reads a plain filename parameter', () {
      expect(categoryFileNameFromQuery('https://example.com/t/1?filename=setup.exe'), 'setup.exe');
      expect(categoryFileNameFromQuery('https://example.com/t/1?filename=setup.exe#frag'), 'setup.exe');
      expect(categoryFileNameFromQuery('https://example.com/t/1?id=1&filename=song.mp3'), 'song.mp3');
    });

    test('reads the filename out of a response header override', () {
      expect(
        categoryFileNameFromQuery(
          'https://example.com/t/1?response-content-disposition=attachment%3B%20filename%3Dgh_2.102.0_windows_amd64.msi',
        ),
        'gh_2.102.0_windows_amd64.msi',
      );
      expect(
        categoryFileNameFromQuery(
          'https://example.com/t/1?response-content-disposition=attachment%3B%20filename%3D%22setup.exe%22',
        ),
        'setup.exe',
      );
    });

    test('returns empty when the query carries no usable name', () {
      expect(categoryFileNameFromQuery('https://example.com/download?id=1'), '');
      expect(categoryFileNameFromQuery('https://example.com/files/setup.exe'), '');
      expect(categoryFileNameFromQuery('https://example.com/t/1?xfilename=setup.exe'), '');
      expect(categoryFileNameFromQuery('https://example.com/t/1?filename='), '');
      expect(categoryFileNameFromQuery('magnet:?xt=urn:btih:abcdef'), '');
    });
  });

  group('matchDownloadCategory', () {
    final categories = [
      DownloadCategory(name: 'Program', path: '/dl/Program', extensions: const ['exe', 'msi']),
      DownloadCategory(name: 'Video', path: '/dl/Video', extensions: const ['mp4']),
      DownloadCategory(name: 'Gone', path: '/dl/Gone', extensions: const ['exe'], isDeleted: true),
      DownloadCategory(name: 'NoPath', path: '   ', extensions: const ['exe']),
    ];

    test('matches the first live category with a path case-insensitively', () {
      expect(matchDownloadCategory(categories, 'SETUP.EXE')?.path, '/dl/Program');
      expect(matchDownloadCategory(categories, 'clip.MP4')?.path, '/dl/Video');
    });

    test('returns null without a matching extension', () {
      expect(matchDownloadCategory(categories, 'archive.rar'), isNull);
      expect(matchDownloadCategory(categories, 'README'), isNull);
    });

    test('built-in categories only match their persisted extensions', () {
      final empty = DownloadCategory(name: '', nameKey: 'categoryProgram', path: '/dl/Program');
      expect(matchDownloadCategory([empty], 'installer.apk'), isNull);
      final seeded = DownloadCategory(
        name: '',
        nameKey: 'categoryProgram',
        path: '/dl/Program',
        extensions: const ['apk'],
      );
      expect(matchDownloadCategory([seeded], 'installer.apk')?.path, '/dl/Program');
      final custom = DownloadCategory(name: 'Custom', path: '/dl/Custom');
      expect(matchDownloadCategory([custom], 'installer.apk'), isNull);
    });
  });

  test('effectiveExtensions returns the stored list and never falls back', () {
    expect(DownloadCategory(name: 'Custom', path: '/x', extensions: const ['zip']).effectiveExtensions(), ['zip']);
    final builtin = DownloadCategory(name: '', nameKey: 'categoryMusic', path: '/m');
    expect(builtin.effectiveExtensions(), isEmpty);
    expect(DownloadCategory(name: 'Custom', path: '/x').effectiveExtensions(), isEmpty);
  });

  test('normalizeCategoryExtensions trims lowercases strips dots dedupes and splits on separators', () {
    expect(normalizeCategoryExtensions([' .MP3 ', 'exe', '.Exe', 'FLAC', 'zip']), ['mp3', 'exe', 'flac', 'zip']);
    expect(normalizeCategoryExtensions(['a b,c']), ['a', 'b', 'c']);
    expect(normalizeCategoryExtensions(['', '  ', '..']), isEmpty);
  });

  test('effectiveExtensions normalizes stored lists in input order', () {
    final stored = DownloadCategory(name: 'X', path: '/x', extensions: const [' .ZIP ', 'Exe', 'exe']);
    expect(stored.effectiveExtensions(), ['zip', 'exe']);
  });

  group('rebaseBuiltinCategoryPaths', () {
    DownloadCategory builtin(String categoryPath, {bool isDeleted = false}) => DownloadCategory(
      name: '',
      nameKey: 'categoryMusic',
      path: categoryPath,
      isBuiltIn: true,
      isDeleted: isDeleted,
      extensions: const ['mp3'],
    );

    test('rebases built-in categories that live under the old directory', () {
      final config = DownloaderConfig()
        ..categories = [
          builtin(path.join('/dl', 'Music')),
          DownloadCategory(name: 'Mine', path: path.join('/dl', 'Mine')),
          DownloadCategory(
            name: 'Elsewhere',
            path: path.join('/other', 'Music'),
            isBuiltIn: true,
            nameKey: 'categoryVideo',
          ),
          builtin(path.join('/dl', 'Old'), isDeleted: true),
        ];

      rebaseBuiltinCategoryPaths(config, oldDir: '/dl', newDir: '/data');

      expect(config.categories[0].path, path.join('/data', 'Music'));
      expect(config.categories[1].path, path.join('/dl', 'Mine'));
      expect(config.categories[2].path, path.join('/other', 'Music'));
      expect(config.categories[3].path, path.join('/dl', 'Old'));
    });

    test('rebases a built-in category that points at the old directory itself', () {
      final config = DownloaderConfig()..categories = [builtin('/dl')];
      rebaseBuiltinCategoryPaths(config, oldDir: '/dl', newDir: '/data');
      expect(config.categories.single.path, '/data');
    });

    test('does nothing when the directories are empty or unchanged', () {
      final config = DownloaderConfig()..categories = [builtin(path.join('/dl', 'Music'))];
      rebaseBuiltinCategoryPaths(config, oldDir: '/dl', newDir: '/dl');
      rebaseBuiltinCategoryPaths(config, oldDir: '', newDir: '/data');
      rebaseBuiltinCategoryPaths(config, oldDir: '/dl', newDir: '');
      expect(config.categories.single.path, path.join('/dl', 'Music'));
    });
  });

  test('categoryFileExtension ignores dots in directory segments', () {
    expect(categoryFileExtension('setup.exe'), 'exe');
    expect(categoryFileExtension('v1.0.zip'), 'zip');
    expect(categoryFileExtension('dir.d/file'), '');
    expect(categoryFileExtension(r'dir.d\file'), '');
    expect(categoryFileExtension('dir/file.exe'), 'exe');
    expect(categoryFileExtension('README'), '');
    expect(categoryFileExtension('trailing.'), '');
  });

  test('categoryFileNameFromUrl percent-decodes the file name', () {
    expect(categoryFileNameFromUrl('https://example.com/my%20file.exe'), 'my file.exe');
    expect(categoryFileNameFromUrl('https://example.com/setup%2Eexe'), 'setup.exe');
    expect(categoryFileNameFromUrl('https://example.com/files/setup.exe?x=1'), 'setup.exe');
    expect(categoryFileNameFromUrl('https://example.com/100%.exe'), '100%.exe');
  });

  test('categoryCandidateFileName mirrors the backend candidate order', () {
    expect(categoryCandidateFileName(rename: 'song.mp3', url: 'https://example.com/real.exe'), 'song.mp3');
    expect(categoryCandidateFileName(rename: 'dir.d/file', url: 'https://example.com/real.exe'), 'real.exe');
    expect(categoryCandidateFileName(rename: 'readme', url: 'https://example.com/real.exe'), 'real.exe');
    expect(categoryCandidateFileName(rename: 'readme', url: 'https://example.com/download?id=1'), '');
  });

  test('go builtinCategoryExtensions stays in sync with kDefaultCategoryExtensions', () {
    File? goSource;
    for (final candidate in ['pkg/base/model.go', '../../pkg/base/model.go']) {
      final file = File(candidate);
      if (file.existsSync()) {
        goSource = file;
        break;
      }
    }
    expect(goSource, isNotNull, reason: 'pkg/base/model.go not found from ${Directory.current}');

    final block = RegExp(
      r'builtinCategoryExtensions = map\[string\]\[\]string\{([\s\S]*?)\n\}',
    ).firstMatch(goSource!.readAsStringSync());
    expect(block, isNotNull, reason: 'builtinCategoryExtensions map literal not found in pkg/base/model.go');

    final entryPattern = RegExp(r'"(\w+)":\s*\{([^}]*)\}');
    final extensionPattern = RegExp(r'"([^"]+)"');
    final goDefaults = <String, Set<String>>{};
    for (final entry in entryPattern.allMatches(block!.group(1)!)) {
      goDefaults[entry.group(1)!] = extensionPattern.allMatches(entry.group(2)!).map((m) => m.group(1)!).toSet();
    }
    expect(goDefaults, isNotEmpty, reason: 'no built-in extension entries parsed from pkg/base/model.go');

    final dartDefaults = kDefaultCategoryExtensions.map((key, value) => MapEntry(key, value.toSet()));
    expect(goDefaults.keys.toSet(), dartDefaults.keys.toSet(), reason: 'built-in category nameKeys drifted');
    for (final key in goDefaults.keys) {
      expect(goDefaults[key], dartDefaults[key], reason: 'extension list drifted for $key');
    }
  });
}
