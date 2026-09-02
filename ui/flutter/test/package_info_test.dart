import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/util/package_info.dart';

void main() {
  group('normalizeAppVersion', () {
    test('treats a four-part platform version as a beta version', () {
      expect(normalizeAppVersion('2.0.0.1'), '2.0.0-beta.1');
      expect(normalizeAppVersion('2.0.0.12'), '2.0.0-beta.12');
    });

    test('keeps stable and semantic prerelease versions unchanged', () {
      expect(normalizeAppVersion('2.0.0'), '2.0.0');
      expect(normalizeAppVersion('2.0.0-beta.1'), '2.0.0-beta.1');
    });
  });
}
