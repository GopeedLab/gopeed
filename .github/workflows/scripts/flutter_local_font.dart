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
//
// Notes:
// - This script DOES NOT call `flutter build web`. It only patches the output.
// - This is a best-effort post-process. Always verify in browser devtools.

import 'dart:async';
import 'dart:io';

Future<void> main(List<String> args) async {
  try {
    // This script is intentionally argument-free for CI convenience.
    // It assumes it is executed from the Flutter project directory (ui/flutter),
    // but will also try to locate pubspec.yaml by walking up.
    final flutterDir = _findFlutterProjectRoot(Directory.current);

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
          p.startsWith('packages/'))
        continue;
      if (p.contains('..')) continue;
      relAssetsUnderS.add(p);
    }

    // Build a download plan: dest relative path (under assets/gstatic/) -> URL.
    final downloads = <String, Uri>{};

    for (final rel in relAssetsUnderS) {
      downloads[rel] = Uri.parse('$gstaticSPrefix$rel');
    }

    if (downloads.isNotEmpty) {
      stdout.writeln(
        'Found ${downloads.length} fonts.gstatic.com assets, downloading...',
      );

      // Reuse a single HttpClient to avoid creating hundreds of short-lived
      // connections (can be flaky on some environments).
      final httpClient =
          HttpClient()
            ..connectionTimeout = const Duration(seconds: 30)
            ..idleTimeout = const Duration(seconds: 30)
            ..maxConnectionsPerHost = 6
            ..userAgent = 'gopeed-ci-gstatic-localizer';
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
