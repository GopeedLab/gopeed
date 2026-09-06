import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import 'app_text_field.dart';

/// A bounded numeric text field with the app's standard spinner controls.
class AppNumberInput extends StatefulWidget {
  const AppNumberInput({
    super.key,
    required this.controller,
    required this.min,
    this.fieldKey,
    this.max,
    this.step = 1,
    this.decimalPlaces = 0,
    this.hintText,
    this.enabled = true,
  });

  final TextEditingController controller;
  final num min;
  final Key? fieldKey;
  final num? max;
  final double step;
  final int decimalPlaces;
  final String? hintText;
  final bool enabled;

  @override
  State<AppNumberInput> createState() => _AppNumberInputState();
}

class _AppNumberInputState extends State<AppNumberInput> {
  bool _normalizing = false;

  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_normalizeDecimalStepValue);
  }

  @override
  void didUpdateWidget(covariant AppNumberInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller == widget.controller) return;
    oldWidget.controller.removeListener(_normalizeDecimalStepValue);
    widget.controller.addListener(_normalizeDecimalStepValue);
  }

  @override
  void dispose() {
    widget.controller.removeListener(_normalizeDecimalStepValue);
    super.dispose();
  }

  void _normalizeDecimalStepValue() {
    if (_normalizing || widget.decimalPlaces <= 0) return;
    final text = widget.controller.text;
    final separator = text.indexOf('.');
    if (separator < 0 || text.length - separator - 1 <= widget.decimalPlaces) return;
    final value = double.tryParse(text);
    if (value == null) return;

    var normalized = value.toStringAsFixed(widget.decimalPlaces);
    normalized = normalized.replaceFirst(RegExp(r'0+$'), '').replaceFirst(RegExp(r'\.$'), '');
    if (normalized == text) return;

    _normalizing = true;
    widget.controller.value = TextEditingValue(
      text: normalized,
      selection: TextSelection.collapsed(offset: normalized.length),
    );
    _normalizing = false;
  }

  @override
  Widget build(BuildContext context) {
    final formatters = widget.decimalPlaces > 0
        ? <TextInputFormatter>[_DecimalNumberFormatter(decimalPlaces: widget.decimalPlaces)]
        : <TextInputFormatter>[
            FilteringTextInputFormatter.digitsOnly,
            _NumericalRangeFormatter(min: widget.min.toInt(), max: widget.max?.toInt()),
          ];
    return AppTextField(
      key: widget.fieldKey,
      controller: widget.controller,
      hintText: widget.hintText,
      keyboardType: widget.decimalPlaces > 0
          ? const TextInputType.numberWithOptions(decimal: true)
          : TextInputType.number,
      inputFormatters: formatters,
      enabled: widget.enabled,
      features: [
        shad.InputFeature.spinner(
          step: widget.step,
          min: widget.min.toDouble(),
          max: widget.max?.toDouble(),
          invalidValue: widget.min.toDouble(),
          enableGesture: false,
        ),
      ],
    );
  }
}

/// Keeps integer text within the same inclusive bounds as the settings UI.
/// Empty text is allowed while editing.
class _NumericalRangeFormatter extends TextInputFormatter {
  const _NumericalRangeFormatter({required this.min, this.max});

  final int min;
  final int? max;

  @override
  TextEditingValue formatEditUpdate(TextEditingValue oldValue, TextEditingValue newValue) {
    if (newValue.text.isEmpty) return newValue;
    final parsed = int.tryParse(newValue.text);
    if (parsed == null) return oldValue;
    final bounded = parsed.clamp(min, max ?? parsed).toInt();
    if (bounded == parsed) return newValue;
    final text = bounded.toString();
    return TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: text.length),
    );
  }
}

/// Allows a non-negative decimal number with a bounded number of fractional digits.
class _DecimalNumberFormatter extends TextInputFormatter {
  _DecimalNumberFormatter({required this.decimalPlaces}) : _pattern = RegExp('^\\d*(?:\\.\\d{0,$decimalPlaces})?\$');

  final int decimalPlaces;
  final RegExp _pattern;

  @override
  TextEditingValue formatEditUpdate(TextEditingValue oldValue, TextEditingValue newValue) {
    return _pattern.hasMatch(newValue.text) ? newValue : oldValue;
  }
}
