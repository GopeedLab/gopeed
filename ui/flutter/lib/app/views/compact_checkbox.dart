import 'package:flutter/material.dart';

class CompactCheckbox extends StatefulWidget {
  final String label;
  final bool value;
  final ValueChanged<bool?>? onChanged;
  final double? scale;
  final TextStyle? textStyle;

  const CompactCheckbox({
    super.key,
    required this.label,
    required this.value,
    this.onChanged,
    this.scale,
    this.textStyle,
  });

  @override
  State<CompactCheckbox> createState() => _CompactCheckboxState();
}

class _CompactCheckboxState extends State<CompactCheckbox> {
  late bool _value;

  @override
  void initState() {
    super.initState();
    _value = widget.value;
  }

  valueChanged(bool? value) {
    setState(() {
      _value = value!;
    });
    widget.onChanged?.call(_value);
  }

  @override
  Widget build(BuildContext context) {
    final checkbox = Checkbox(
      value: _value,
      onChanged: valueChanged,
    );

    return TextButton(
      onPressed: () {
        valueChanged(!_value);
      },
      child: Row(
        children: [
          widget.scale == null
              ? checkbox
              : Transform.scale(
                  scale: widget.scale,
                  child: checkbox,
                ),
          Text(
            widget.label,
            style: widget.textStyle,
          ),
        ],
      ),
    );
  }
}
