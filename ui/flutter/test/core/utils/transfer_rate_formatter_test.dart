import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/utils/transfer_rate_formatter.dart';

void main() {
  test('formats transfer rates with adaptive binary units', () {
    expect(TransferRateFormatter.format(-1).text, '0 B/s');
    expect(TransferRateFormatter.format(512).text, '512 B/s');
    expect(TransferRateFormatter.format(96 * 1024).text, '96 KB/s');
    expect(TransferRateFormatter.format(1536).text, '1.5 KB/s');
    expect(TransferRateFormatter.format(1280).text, '1.25 KB/s');
    expect(TransferRateFormatter.format(10 * 1024 * 1024).text, '10 MB/s');
    expect(TransferRateFormatter.format(2 * 1024 * 1024 * 1024).text, '2 GB/s');
  });
}
