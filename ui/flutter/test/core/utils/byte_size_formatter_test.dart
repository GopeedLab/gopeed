import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/utils/byte_size_formatter.dart';

void main() {
  test('formats scaled byte sizes with up to two decimal places', () {
    expect(ByteSizeFormatter.format(-1), '0 B');
    expect(ByteSizeFormatter.format(512), '512 B');
    expect(ByteSizeFormatter.format(1536), '1.5 KB');
    expect(ByteSizeFormatter.format(1280), '1.25 KB');
    expect(ByteSizeFormatter.format(32 * 1024 * 1024), '32 MB');
    expect(ByteSizeFormatter.format(2 * 1024 * 1024 * 1024), '2 GB');
  });
}
