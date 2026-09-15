import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/utils/compact_count_formatter.dart';

void main() {
  test('keeps small counts exact and compacts larger counts to a bounded label', () {
    expect(CompactCountFormatter.format(-1), '0');
    expect(CompactCountFormatter.format(0), '0');
    expect(CompactCountFormatter.format(9999), '9999');
    expect(CompactCountFormatter.format(10000), '1w');
    expect(CompactCountFormatter.format(12345), '1.2w');
    expect(CompactCountFormatter.format(99999), '9.9w');
    expect(CompactCountFormatter.format(123456), '12w');
    expect(CompactCountFormatter.format(1000000), '1m');
    expect(CompactCountFormatter.format(1234567), '1.2m');
    expect(CompactCountFormatter.format(10000000), '10m');
    expect(CompactCountFormatter.format(12345678), '12m');
    expect(CompactCountFormatter.format(100000000), '100m');
    expect(CompactCountFormatter.format(123456789), '123m');
    expect(CompactCountFormatter.format(1000000000), '1b');
    expect(CompactCountFormatter.format(1234567890), '1.2b');
    expect(CompactCountFormatter.format(12345678901), '12b');
    expect(CompactCountFormatter.format(1234567890123), '1.2t');
    expect(CompactCountFormatter.format(1234567890123456), '1.2q');
    expect(CompactCountFormatter.format(9223372036854775807), '9.2qi');
  });
}
