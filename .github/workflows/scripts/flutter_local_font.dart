// Localize external Google Fonts (fonts.gstatic.com) references in Flutter Web build output.
//
// Usage (post-build):
//   # Run AFTER: flutter build web --no-web-resources-cdn
//   dart ../../.github/workflows/scripts/flutter_local_font.dart
//
// What it does:
// In main.dart.js:
//    - Find any referenced resources under: https://fonts.gstatic.com/s/
//      (including the common pattern where Flutter concatenates the base URL with a relative path)
//    - Download them into: build/web/assets/gstatic/<path>
//    - Rewrite "https://fonts.gstatic.com/s/" -> "assets/gstatic/" so runtime loads locally
//    - Only download font subsets needed by supported locales (parsed from message.dart)
//
// Notes:
// - This script DOES NOT call `flutter build web`. It only patches the output.
// - This is a best-effort post-process. Always verify in browser devtools.

import 'dart:async';
import 'dart:io';

/// Get font families required for a specific locale based on language/script.
/// Uses language code prefix to automatically detect the script system.
/// Returns empty set if base fonts (Latin/Cyrillic/Greek) are sufficient.
Set<String> _getFontFamiliesForLocale(String locale) {
  final lang = locale.toLowerCase().split('_').first;

  // CJK languages - need specific large font files
  if (lang == 'zh') {
    // Simplified vs Traditional Chinese
    if (locale.contains('cn') || locale.contains('sg')) {
      return {'notosanssc'}; // Simplified Chinese
    }
    return {'notosanstc', 'notosanshk'}; // Traditional Chinese (TW, HK, MO)
  }
  if (lang == 'ja') return {'notosansjp'}; // Japanese
  if (lang == 'ko') return {'notosanskr'}; // Korean

  // Arabic script languages
  if (lang == 'ar') return {'notosansarabic'}; // Arabic
  if (lang == 'fa') return {'notosansarabic'}; // Persian/Farsi
  if (lang == 'ur') return {'notosansarabic'}; // Urdu
  if (lang == 'ps') return {'notosansarabic'}; // Pashto
  if (lang == 'ku') return {'notosansarabic'}; // Kurdish (Arabic script)

  // Hebrew script
  if (lang == 'he' || lang == 'yi') return {'notosanshebrew'};

  // South Asian scripts
  if (lang == 'hi' || lang == 'mr' || lang == 'ne' || lang == 'sa') {
    return {'notosansdevanagari'}; // Hindi, Marathi, Nepali, Sanskrit
  }
  if (lang == 'bn' || lang == 'as')
    return {'notosansbengali'}; // Bengali, Assamese
  if (lang == 'ta')
    return {'notosanstamil', 'notosanstamilsupplement'}; // Tamil
  if (lang == 'te') return {'notosanstelugu'}; // Telugu
  if (lang == 'kn') return {'notosanskannada'}; // Kannada
  if (lang == 'ml') return {'notosansmalayalam'}; // Malayalam
  if (lang == 'gu') return {'notosansgujarati'}; // Gujarati
  if (lang == 'pa') return {'notosansgurmukhi'}; // Punjabi (Gurmukhi)
  if (lang == 'or') return {'notosansoriya'}; // Odia/Oriya
  if (lang == 'si') return {'notosanssinhala'}; // Sinhala

  // Southeast Asian scripts
  if (lang == 'th') return {'notosansthai'}; // Thai
  if (lang == 'lo') return {'notosanslao'}; // Lao
  if (lang == 'my') return {'notosansmyanmar'}; // Myanmar/Burmese
  if (lang == 'km') return {'notosanskhmer'}; // Khmer/Cambodian
  if (lang == 'jv') return {'notosansjavanese'}; // Javanese

  // Other scripts
  if (lang == 'ka') return {'notosansgeorgian'}; // Georgian
  if (lang == 'hy') return {'notosansarmenian'}; // Armenian
  if (lang == 'am' || lang == 'ti')
    return {'notosansethiopic'}; // Amharic, Tigrinya
  if (lang == 'mn') return {'notosansmongolian'}; // Mongolian

  // Latin/Cyrillic/Greek based languages - base notosans is sufficient
  // Includes: English, German, French, Spanish, Italian, Portuguese,
  // Russian, Ukrainian, Polish, Turkish, Vietnamese, Greek, etc.
  return {};
}

/// Base font families that are always included regardless of locale.
/// These provide core functionality and are relatively small.
const _baseFontFamilies = <String>{
  'notosans', // Base Latin/Cyrillic/Greek/Vietnamese
  'roboto', // Material Design default font
  'notosanssymbols', // Common symbols
  'notosanssymbols2', // Additional symbols
  'notosansmath', // Math symbols
  'notomusic', // Music notation symbols
  // Note: notocoloremoji (~24MB) and notoemoji (~860KB) are excluded
  // to reduce bundle size. Add them back if emoji support is needed.
};

/// Parse supported locales from message.dart file.
/// Reads the import statements and extracts locale codes like 'zh_cn', 'en_us', etc.
Set<String> _parseSupportedLocales(File messageFile) {
  final locales = <String>{};

  if (!messageFile.existsSync()) {
    _fail('Warning: message.dart not found, only base fonts will be used');
  }

  final content = messageFile.readAsStringSync();

  // Match import statements like: import 'langs/zh_cn.dart';
  final importRegex = RegExp(r"import\s+'langs/(\w+)\.dart'");
  for (final match in importRegex.allMatches(content)) {
    final locale = match.group(1);
    if (locale != null) {
      locales.add(locale);
    }
  }

  if (locales.isEmpty) {
    _fail(
        'Warning: No locales found in message.dart, only base fonts will be used');
  }

  return locales;
}

/// Get required font families for the given locales.
/// Returns a set of font family names that should be included.
/// Automatically detects required fonts based on language code prefix.
Set<String> _getRequiredFontFamilies(Set<String> locales) {
  final families = <String>{..._baseFontFamilies};
  for (final locale in locales) {
    final localeFamilies = _getFontFamiliesForLocale(locale);
    families.addAll(localeFamilies);
  }
  return families;
}

/// Check if a font file path should be included based on required font families.
/// Font paths have the format: "fontfamily/version/filename.ext"
/// Example: "notosanssc/v36/xxx.ttf" -> font family is "notosanssc"
bool _fontMatchesRequiredFamilies(
    String fontPath, Set<String> requiredFamilies) {
  // Extract font family from path (first segment before '/')
  final slashIndex = fontPath.indexOf('/');
  if (slashIndex == -1) {
    // No slash found, include by default (unusual path format)
    return true;
  }

  final fontFamily = fontPath.substring(0, slashIndex).toLowerCase();
  return requiredFamilies.contains(fontFamily);
}

Future<void> main(List<String> args) async {
  try {
    // This script is intentionally argument-free for CI convenience.
    // It assumes it is executed from the Flutter project directory (ui/flutter),
    // but will also try to locate pubspec.yaml by walking up.
    final flutterDir = _findFlutterProjectRoot(Directory.current);

    // Parse supported locales from message.dart
    final messageFile =
        File(_join(flutterDir.path, 'lib', 'i18n', 'message.dart'));
    final supportedLocales = _parseSupportedLocales(messageFile);
    stdout.writeln('Supported locales: ${supportedLocales.join(', ')}');

    // Get required font families based on supported locales
    final requiredFamilies = _getRequiredFontFamilies(supportedLocales);
    stdout.writeln('Required font families: ${requiredFamilies.join(', ')}');

    final webDir = Directory(_join(flutterDir.path, 'build', 'web'));
    if (!webDir.existsSync()) {
      _fail(
        'Web build directory not found: ${webDir.path}\n'
        'Did you run: flutter build web --no-web-resources-cdn ?',
      );
    }

    final mainJs = File(_join(webDir.path, 'main.dart.js'));
    if (!mainJs.existsSync()) {
      _fail('main.dart.js not found at: ${mainJs.path}');
    }

    final gstaticRoot = Directory(_join(webDir.path, 'assets', 'gstatic'));
    if (!gstaticRoot.existsSync()) gstaticRoot.createSync(recursive: true);

    const gstaticSPrefix = 'https://fonts.gstatic.com/s/';
    final original = mainJs.readAsStringSync();

    // Relative font asset paths which Flutter's loader typically concatenates with:
    //   https://fonts.gstatic.com/s/
    // Keep this as a best-effort heuristic to cover the common concatenation pattern.
    // Some gstatic assets include extra dot segments like: notocoloremoji/v32/Yq6P-KqIXTD0t4D9z1ESnKM3-HpFabsE4tq3luCC7p-aXxcn.0.woff2
    // Allow dots in the path, while still restricting to font-like extensions.
    final relAssetRegex = RegExp(
      r"""["']([a-zA-Z0-9/_\.-]+\.(?:woff2|woff|ttf|otf|eot|svg))["']""",
      caseSensitive: false,
    );
    final relAssetsUnderS = <String>{};
    for (final m in relAssetRegex.allMatches(original)) {
      final p = m.group(1);
      if (p == null || p.isEmpty) continue;
      if (p.contains('://') || p.startsWith('data:')) continue;
      if (p.startsWith('/') ||
          p.startsWith('assets/') ||
          p.startsWith('packages/')) continue;
      if (p.contains('..')) continue;
      relAssetsUnderS.add(p);
    }

    if (relAssetsUnderS.isEmpty) {
      _fail('No fonts.gstatic.com assets found in main.dart.js.');
    }

    // Build a download plan: dest relative path (under assets/gstatic/) -> URL.
    // Filter fonts based on required font families to reduce package size.
    final downloads = <String, Uri>{};
    final skippedFonts = <String, Set<String>>{};

    for (final rel in relAssetsUnderS) {
      if (_fontMatchesRequiredFamilies(rel, requiredFamilies)) {
        downloads[rel] = Uri.parse('$gstaticSPrefix$rel');
      } else {
        // Track skipped fonts and their paths for fallback generation
        final slashIndex = rel.indexOf('/');
        final family = slashIndex > 0 ? rel.substring(0, slashIndex) : rel;
        skippedFonts.putIfAbsent(family, () => <String>{}).add(rel);
      }
    }

    if (skippedFonts.isNotEmpty) {
      stdout.writeln(
        'Skipped ${skippedFonts.length} font families not required by supported locales:',
      );
      stdout.writeln('  ${skippedFonts.keys.join(', ')}');
    }

    if (downloads.isNotEmpty) {
      stdout.writeln(
        'Found ${downloads.length} fonts.gstatic.com assets to download...',
      );

      // Reuse a single HttpClient to avoid creating hundreds of short-lived
      // connections (can be flaky on some environments).
      final httpClient = HttpClient()
        ..connectionTimeout = const Duration(seconds: 30)
        ..idleTimeout = const Duration(seconds: 30)
        ..maxConnectionsPerHost = 6;
      try {
        for (final entry in downloads.entries) {
          final relPath = entry.key;
          final url = entry.value;
          final destPath = relPath.replaceAll('/', Platform.pathSeparator);
          final dest = File(_join(gstaticRoot.path, destPath));
          await _downloadIfMissing(httpClient, url, dest);
        }
      } finally {
        httpClient.close(force: true);
      }
    }

    // Generate empty fallback font files for skipped fonts to prevent infinite loading
    // Empty files will cause browsers to fail fast instead of retrying indefinitely
    if (skippedFonts.isNotEmpty) {
      var fallbackCount = 0;
      for (final entry in skippedFonts.entries) {
        for (final relPath in entry.value) {
          final destPath = relPath.replaceAll('/', Platform.pathSeparator);
          final dest = File(_join(gstaticRoot.path, destPath));
          if (!dest.existsSync()) {
            dest.parent.createSync(recursive: true);
            // Create an empty file - browsers will recognize it as invalid and skip
            dest.writeAsBytesSync([]);
            fallbackCount++;
          }
        }
      }
      if (fallbackCount > 0) {
        stdout.writeln(
          'Generated $fallbackCount empty fallback font files to prevent infinite loading',
        );
      }
    }

    // Rewrite remote prefix so requests become:
    //   <origin>/assets/gstatic/<...>.woff2
    final replacedGstatic = original.replaceAll(
      gstaticSPrefix,
      'assets/gstatic/',
    );

    if (replacedGstatic == original && downloads.isEmpty) {
      stdout.writeln(
        'No changes applied (no fonts.gstatic.com references found).',
      );
      exitCode = 0;
      return;
    }

    mainJs.writeAsStringSync(replacedGstatic);

    stdout.writeln('Patched: ${mainJs.path}');
    if (downloads.isNotEmpty) {
      stdout.writeln(
        ' - downloaded ${downloads.length} files into: ${gstaticRoot.path}',
      );
    }
    stdout.writeln(' - fonts.gstatic.com rewritten -> assets/gstatic/');
  } catch (e, st) {
    stderr.writeln('ERROR: $e');
    stderr.writeln(st);
    exitCode = 1;
  }
}

Future<void> _downloadIfMissing(HttpClient client, Uri url, File dest) async {
  if (dest.existsSync() && dest.lengthSync() > 0) return;

  dest.parent.createSync(recursive: true);

  // Network can be flaky in CI. Retry a few times on transient errors.
  const maxAttempts = 4;

  for (var attempt = 1; attempt <= maxAttempts; attempt++) {
    final tmp = File('${dest.path}.tmp');
    try {
      try {
        final req = await client.getUrl(url);

        final resp = await req.close().timeout(const Duration(seconds: 60));
        if (resp.statusCode != 200) {
          // 404 is not transient in practice - fail fast with a clear message.
          if (resp.statusCode == 404) {
            _fail('Missing gstatic asset (404): $url');
          }
          throw HttpException('HTTP ${resp.statusCode}', uri: url);
        }

        // Stream to temp file first, then rename to avoid leaving partial files.
        if (tmp.existsSync()) tmp.deleteSync();
        final sink = tmp.openWrite();
        await resp.pipe(sink).timeout(const Duration(seconds: 120));

        if (!tmp.existsSync() || tmp.lengthSync() == 0) {
          throw const FormatException('Downloaded asset is empty');
        }
        if (dest.existsSync()) dest.deleteSync();
        tmp.renameSync(dest.path);
        return;
      } finally {
        // no-op: HttpClient lifecycle managed by the caller
      }
    } on TimeoutException {
      if (tmp.existsSync()) tmp.deleteSync();
      if (attempt == maxAttempts) {
        _fail('Timeout downloading asset: $url');
      }
    } on SocketException catch (e) {
      if (tmp.existsSync()) tmp.deleteSync();
      if (attempt == maxAttempts) {
        _fail('Socket error downloading asset: $url ($e)');
      }
    } on HttpException catch (e) {
      if (tmp.existsSync()) tmp.deleteSync();
      if (attempt == maxAttempts) {
        _fail('HTTP error downloading asset: $url ($e)');
      }
    } on FileSystemException catch (e) {
      if (tmp.existsSync()) tmp.deleteSync();
      _fail('File write error downloading asset: $url ($e)');
    } catch (e) {
      if (tmp.existsSync()) tmp.deleteSync();
      if (attempt == maxAttempts) {
        _fail('Unexpected error downloading asset: $url ($e)');
      }
    }

    // Backoff before retrying.
    final backoffSeconds = 1 << (attempt - 1);
    stdout.writeln(
      'Retrying (${attempt + 1}/$maxAttempts) for: $url (wait ${backoffSeconds}s)',
    );
    await Future.delayed(Duration(seconds: backoffSeconds));
  }
}

Directory _findFlutterProjectRoot(Directory start) {
  var dir = start;
  for (var i = 0; i < 10; i++) {
    final pubspec = File(_join(dir.path, 'pubspec.yaml'));
    if (pubspec.existsSync()) return dir;
    final parent = dir.parent;
    if (parent.path == dir.path) break;
    dir = parent;
  }
  // Fallback: use current directory, error messages will explain what is missing.
  return start;
}

Never _fail(String msg) {
  throw FormatException(msg);
}

String _join(String a, [String? b, String? c, String? d]) {
  final parts = <String>[
    a,
    if (b != null) b,
    if (c != null) c,
    if (d != null) d,
  ];
  return parts.join(Platform.pathSeparator);
}
