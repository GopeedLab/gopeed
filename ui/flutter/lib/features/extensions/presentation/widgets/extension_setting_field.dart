import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter/services.dart';

import '../../../../api/model/extension.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_text_field.dart';

class ExtensionSettingField extends StatelessWidget {
  const ExtensionSettingField({super.key, required this.setting, required this.controller});

  final Setting setting;
  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          setting.title,
          style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w700),
        ),
        if (setting.description.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(setting.description, style: TextStyle(color: palette.textSecondary, fontSize: 12, height: 1.35)),
        ],
        const SizedBox(height: 8),
        _buildInput(),
      ],
    );
  }

  Widget _buildInput() {
    if (setting.type == SettingType.boolean) {
      return ValueListenableBuilder<TextEditingValue>(
        valueListenable: controller,
        builder: (context, value, _) =>
            Switch(value: value.text == 'true', onChanged: (enabled) => controller.text = enabled.toString()),
      );
    }

    final options = setting.options;
    if (options != null && options.isNotEmpty) {
      return ValueListenableBuilder<TextEditingValue>(
        valueListenable: controller,
        builder: (context, value, _) => LayoutBuilder(
          builder: (context, constraints) => Select<String>(
            value: options.any((option) => option.value.toString() == value.text) ? value.text : null,
            placeholder: Text(setting.title),
            constraints: BoxConstraints.tightFor(width: constraints.maxWidth),
            popupConstraints: const BoxConstraints(maxHeight: 240),
            itemBuilder: (context, selectedValue) => Text(
              options.firstWhere((option) => option.value.toString() == selectedValue).label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            popup: (context) => SelectPopup<String>(
              items: SelectItemList(
                children: [
                  for (final option in options)
                    SelectItemButton<String>(value: option.value.toString(), child: Text(option.label)),
                ],
              ),
            ),
            onChanged: (selectedValue) {
              if (selectedValue != null) controller.text = selectedValue;
            },
          ),
        ),
      );
    }

    final isNumber = setting.type == SettingType.number;
    return AppTextField(
      controller: controller,
      keyboardType: isNumber ? const TextInputType.numberWithOptions(decimal: true, signed: true) : TextInputType.text,
      inputFormatters: isNumber
          ? [
              TextInputFormatter.withFunction(
                (oldValue, newValue) => RegExp(r'^-?\d*\.?\d*$').hasMatch(newValue.text) ? newValue : oldValue,
              ),
            ]
          : null,
    );
  }
}
