import 'package:flutter/widgets.dart';

import '../../core/utils/breakpoints.dart';
import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';

class AppChoiceOption<T extends Object> {
  const AppChoiceOption({required this.value, required this.label, required this.icon, this.key});

  final T value;
  final String label;
  final IconData icon;
  final String? key;
}

class AppChoiceSegmentedControl<T extends Object> extends StatelessWidget {
  const AppChoiceSegmentedControl({
    super.key,
    required this.value,
    required this.options,
    required this.onChanged,
    this.buttonKeyPrefix = 'choice',
    this.alignment,
  });

  final T value;
  final List<AppChoiceOption<T>> options;
  final ValueChanged<T> onChanged;
  final String buttonKeyPrefix;
  final WrapAlignment? alignment;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      alignment: alignment ?? (desktop ? WrapAlignment.end : WrapAlignment.start),
      children: [
        for (final option in options)
          _ChoiceButton<T>(
            key: ValueKey('$buttonKeyPrefix-${option.key ?? option.value}'),
            option: option,
            selected: option.value == value,
            palette: palette,
            onPressed: () => onChanged(option.value),
          ),
      ],
    );
  }
}

class _ChoiceButton<T extends Object> extends StatelessWidget {
  const _ChoiceButton({
    super.key,
    required this.option,
    required this.selected,
    required this.palette,
    required this.onPressed,
  });

  final AppChoiceOption<T> option;
  final bool selected;
  final AppPalette palette;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: onPressed,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 140),
        height: 34,
        padding: const EdgeInsets.symmetric(horizontal: 10),
        decoration: BoxDecoration(
          color: selected ? palette.itemActiveBg : palette.cardBg,
          borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
          border: Border.all(color: selected ? palette.textPrimary : palette.border),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(option.icon, size: 15, color: selected ? palette.textPrimary : palette.textSecondary),
            const SizedBox(width: 7),
            Text(
              option.label,
              style: TextStyle(
                color: selected ? palette.textPrimary : palette.textSecondary,
                fontSize: 12,
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
