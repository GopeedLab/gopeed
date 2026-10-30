import 'dart:convert';
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

  test('BitTorrent peer discovery settings round-trip with the Go json keys', () {
    final config = DownloaderConfig();
    config.protocolConfig.bt
      ..disableDht = true
      ..disablePex = true;

    final json = config.toJson();
    final bt = (json['protocolConfig'] as Map<String, dynamic>)['bt'] as Map<String, dynamic>;
    expect(bt['disableDht'], isTrue);
    expect(bt['disablePex'], isTrue);

    final decoded = DownloaderConfig.fromJson(json).protocolConfig.bt;
    expect(decoded.disableDht, isTrue);
    expect(decoded.disablePex, isTrue);
  });

  test('FTP settings round-trip with the Go json keys', () {
    final config = DownloaderConfig();
    config.protocolConfig.ftp.connections = 6;

    final json = config.toJson();
    final ftp = (json['protocolConfig'] as Map<String, dynamic>)['ftp'] as Map<String, dynamic>;
    expect(ftp, {'connections': 6});
    expect(DownloaderConfig.fromJson(json).protocolConfig.ftp.connections, 6);
  });

  test('a stored config without FTP settings still parses', () {
    final json = DownloaderConfig().toJson();
    (json['protocolConfig'] as Map<String, dynamic>).remove('ftp');

    final decoded = DownloaderConfig.fromJson(json);
    expect(decoded.protocolConfig.ftp.connections, 0);
    expect(decoded.protocolConfig.http.connections, DownloaderConfig().protocolConfig.http.connections);
  });

  test('BitTorrent peer discovery stays on when a stored config lacks the keys', () {
    final bt = BtConfig.fromJson({'listenPort': 6881, 'seedRatio': 1.0});
    expect(bt.disableDht, isFalse);
    expect(bt.disablePex, isFalse);
  });

  test('Flutter preferences round-trip through the Go-owned extra config', () {
    final config = DownloaderConfig();
    config.extra
      ..windowState = WindowStateConfig(isMaximized: true, width: 1280, height: 720)
      ..bookmarks = {'downloads': 'D:/Downloads'}
      ..createHistory = ['https://example.com/file.zip']
      ..runAsMenubarApp = true
      ..analyticsEnabled = false
      ..deviceId = 'client-id';

    final decoded = DownloaderConfig.fromJson(config.toJson()).extra;
    expect(decoded.windowState.isMaximized, isTrue);
    expect(decoded.windowState.width, 1280);
    expect(decoded.windowState.height, 720);
    expect(decoded.bookmarks, {'downloads': 'D:/Downloads'});
    expect(decoded.createHistory, ['https://example.com/file.zip']);
    expect(decoded.runAsMenubarApp, isTrue);
    expect(decoded.analyticsEnabled, isFalse);
    expect(decoded.deviceId, 'client-id');
  });

  test('legacy analytics client IDs are ignored without migration', () {
    final config = ExtraConfig.fromJson({'analyticsClientId': 'AP3A.241105.008'});

    expect(config.deviceId, isEmpty);
    expect(config.toJson(), isNot(contains('analyticsClientId')));
  });

  test('analytics fallback ID uses only its dedicated config field', () {
    final config = ExtraConfig.fromJson({'analyticsClientId': 'AP3A.241105.008', 'deviceId': 'stored-fallback'});

    expect(config.deviceId, 'stored-fallback');
    expect(config.toJson()['deviceId'], 'stored-fallback');
    expect(config.toJson(), isNot(contains('analyticsClientId')));
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
    expect(DownloaderConfig.fromJson(missing).autoCategorize, isFalse);

    // The Go backend marshals a nil category slice and an unset switch as null.
    final nil = DownloaderConfig().toJson()
      ..['categories'] = null
      ..['autoCategorize'] = null;
    expect(DownloaderConfig.fromJson(nil).categories, isEmpty);
    expect(DownloaderConfig.fromJson(nil).autoCategorize, isFalse);
  });

  test('auto categorize can be turned off and round-trips', () {
    final config = DownloaderConfig(autoCategorize: false);
    expect(DownloaderConfig.fromJson(config.toJson()).autoCategorize, isFalse);
  });

  test('auto categorize is off unless the config turns it on', () {
    expect(DownloaderConfig().autoCategorize, isFalse);
    expect(DownloaderConfig(autoCategorize: true).autoCategorize, isTrue);
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

    test('rebases a windows path that the backend seeded with a backslash', () {
      // The backend seeds the categories with the separator of its platform.
      final config = DownloaderConfig()
        ..categories = [
          builtin(r'C:\Users\me\Downloads\Music'),
          DownloadCategory(name: 'Mine', path: r'C:\Users\me\Downloads\Mine'),
        ];

      rebaseBuiltinCategoryPaths(config, oldDir: r'C:\Users\me\Downloads', newDir: r'C:\Users\me\New');

      // The joined separator depends on the platform the test runs on, so the
      // assertion compares the forward slash form.
      expect(config.categories[0].path.replaceAll(r'\', '/'), 'C:/Users/me/New/Music');
      expect(config.categories[1].path, r'C:\Users\me\Downloads\Mine');
    });
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

  test('runs the extension rule table shared with the backend', () {
    // pkg/download/testdata/category_matching_cases.json is also read by the Go
    // test, so a rule that only changes on one side breaks the other side.
    final cases = jsonDecode(_readSharedRuleTable()) as Map<String, dynamic>;
    for (final c in cases['normalizeExtensions'] as List) {
      final input = (c['input'] as List).cast<String>();
      final expected = (c['expected'] as List).cast<String>();
      expect(normalizeCategoryExtensions(input), expected, reason: 'normalizeCategoryExtensions($input)');
    }
  });
}

/// The rule table is shared with the backend, see
/// pkg/download/testdata/category_matching_cases.json.
String _readSharedRuleTable() {
  for (final candidate in [
    'pkg/download/testdata/category_matching_cases.json',
    '../../pkg/download/testdata/category_matching_cases.json',
  ]) {
    final file = File(candidate);
    if (file.existsSync()) {
      return file.readAsStringSync();
    }
  }
  throw StateError('category_matching_cases.json not found from ${Directory.current}');
}
