class FormattedTransferRate {
  const FormattedTransferRate({required this.value, required this.unit});

  final String value;
  final String unit;

  String get text => '$value $unit';
}

abstract final class TransferRateFormatter {
  static const _units = ['B/s', 'KB/s', 'MB/s', 'GB/s', 'TB/s'];

  static FormattedTransferRate format(int bytesPerSecond) {
    var value = bytesPerSecond < 0 ? 0.0 : bytesPerSecond.toDouble();
    var unitIndex = 0;
    while (value >= 1024 && unitIndex < _units.length - 1) {
      value /= 1024;
      unitIndex += 1;
    }

    final formattedValue = unitIndex == 0 ? value.round().toString() : _trimFraction(value.toStringAsFixed(2));
    return FormattedTransferRate(value: formattedValue, unit: _units[unitIndex]);
  }

  static String _trimFraction(String value) {
    return value.replaceFirst(RegExp(r'0+$'), '').replaceFirst(RegExp(r'\.$'), '');
  }
}
