import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/utils/duration_formatter.dart';

void main() {
  test('formats download durations with stable clock notation', () {
    expect(DurationFormatter.format(Duration.zero), '0:00');
    expect(DurationFormatter.format(const Duration(seconds: 65)), '1:05');
    expect(DurationFormatter.format(const Duration(hours: 27, minutes: 2, seconds: 3)), '27:02:03');
  });

  test('rounds sub-second download durations up', () {
    expect(DurationFormatter.format(const Duration(microseconds: 1)), '0:01');
  });
}
