// Validate Flutter ARB translation catalogs before localization code generation.
//
// Usage (from ui/flutter):
//   dart run ../../.github/workflows/scripts/check_l10n.dart
//
// What it checks:
// - Every ARB file uses the repository's canonical two-space JSON formatting.
// - Every app_<locale>.arb file declares the matching @@locale value.
// - Message values are not empty.
// - Every locale has exactly the same message keys as app_en.arb.
// - Every translation preserves the placeholder variables used by app_en.arb.
//
// Notes:
// - This script does not judge translation quality.
// - It does not generate localization code or fully parse ICU MessageFormat.
//   `flutter gen-l10n` runs immediately afterwards in CI to validate ICU syntax
//   and generate app_localizations*.dart.

import 'dart:convert';
import 'dart:io';

/// Runs repository-specific checks that `flutter gen-l10n` does not enforce.
void main() {
  final directory = Directory('lib/l10n');
  final templateFile = File('${directory.path}/app_en.arb');
  final template = _readArb(templateFile);
  final templateKeys = _messageKeys(template);
  var failed = false;

  for (final file in directory.listSync().whereType<File>().where(
    (file) => file.path.endsWith('.arb'),
  )) {
    final messages = _readArb(file);
    if (!_hasCanonicalFormat(file, messages)) {
      stderr.writeln(
        '${file.path}: ARB formatting is not canonical (use two-space JSON indentation and a trailing newline)',
      );
      failed = true;
    }
    final expectedLocale = file.uri.pathSegments.last
        .replaceFirst('app_', '')
        .replaceFirst('.arb', '');
    if (messages['@@locale'] != expectedLocale) {
      stderr.writeln('${file.path}: @@locale must be $expectedLocale');
      failed = true;
    }
    final keys = _messageKeys(messages);
    final empty =
        keys.where((key) => (messages[key] as String).trim().isEmpty).toList()
          ..sort();
    if (empty.isNotEmpty) {
      stderr.writeln('${file.path}: empty messages $empty');
      failed = true;
    }
    if (file.path == templateFile.path) continue;
    final missing = templateKeys.difference(keys);
    final extra = keys.difference(templateKeys);
    if (missing.isNotEmpty || extra.isNotEmpty) {
      stderr.writeln(
        '${file.path}: missing ${missing.toList()..sort()}, extra ${extra.toList()..sort()}',
      );
      failed = true;
    }

    for (final key in templateKeys.intersection(keys)) {
      final expected = _placeholders(template[key] as String);
      final actual = _placeholders(messages[key] as String);
      if (!_sameSet(expected, actual)) {
        stderr.writeln(
          '${file.path}: $key placeholders $actual do not match $expected',
        );
        failed = true;
      }
    }
  }

  if (failed) exitCode = 1;
}

/// Reads one ARB file as its top-level JSON object.
Map<String, Object?> _readArb(File file) =>
    jsonDecode(file.readAsStringSync()) as Map<String, Object?>;

/// Checks the exact JSON layout so whitespace-only formatting drift fails CI.
bool _hasCanonicalFormat(File file, Map<String, Object?> arb) {
  const encoder = JsonEncoder.withIndent('  ');
  return file.readAsStringSync() == '${encoder.convert(arb)}\n';
}

/// Returns user-facing message keys and excludes ARB metadata keys (`@`/`@@`).
Set<String> _messageKeys(Map<String, Object?> arb) =>
    arb.keys.where((key) => !key.startsWith('@')).toSet();

/// Extracts both simple placeholders (`{name}`) and ICU selector variables.
///
/// For example, this returns `count` from
/// `{count, plural, =1{1 file} other{{count} files}}`.
Set<String> _placeholders(String message) => RegExp(
  r'\{([A-Za-z_][A-Za-z0-9_]*)(?=\s*(?:\}|,\s*(?:plural|select|selectordinal)\s*,))',
).allMatches(message).map((match) => match.group(1)!).toSet();

/// Compares sets without depending on iteration order.
bool _sameSet(Set<String> left, Set<String> right) =>
    left.length == right.length && left.containsAll(right);
