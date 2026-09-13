abstract final class CompactCountFormatter {
  // Counts start compacting at ten thousand by product convention, then use
  // progressively larger decimal units so the rendered label stays bounded.
  static const _units = [
    (divisor: 1000000000000000000, suffix: 'qi'),
    (divisor: 1000000000000000, suffix: 'q'),
    (divisor: 1000000000000, suffix: 't'),
    (divisor: 1000000000, suffix: 'b'),
    (divisor: 1000000, suffix: 'm'),
    (divisor: 10000, suffix: 'w'),
  ];

  static String format(int count) {
    if (count <= 0) return '0';
    if (count < _units.last.divisor) return count.toString();

    final unit = _units.firstWhere((unit) => count >= unit.divisor);
    final whole = count ~/ unit.divisor;
    final tenth = (count % unit.divisor) ~/ (unit.divisor ~/ 10);
    final value = whole < 10 && tenth > 0 ? '$whole.$tenth' : '$whole';
    return '$value${unit.suffix}';
  }
}
