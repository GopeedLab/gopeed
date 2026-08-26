abstract final class ByteSizeFormatter {
  static const _units = ['B', 'KB', 'MB', 'GB', 'TB'];

  static String format(int bytes) {
    if (bytes <= 0) return '0 B';

    var value = bytes.toDouble();
    var unitIndex = 0;
    while (value >= 1024 && unitIndex < _units.length - 1) {
      value /= 1024;
      unitIndex++;
    }

    // Bytes are whole values. Scaled binary units keep at most two decimal
    // places and omit trailing zeroes (for example: 1.5 MB, 1.25 MB, 2 MB).
    final formattedValue = unitIndex == 0 ? value.round().toString() : _trimFraction(value.toStringAsFixed(2));
    return '$formattedValue ${_units[unitIndex]}';
  }

  static String _trimFraction(String value) {
    return value.replaceFirst(RegExp(r'0+$'), '').replaceFirst(RegExp(r'\.$'), '');
  }
}
