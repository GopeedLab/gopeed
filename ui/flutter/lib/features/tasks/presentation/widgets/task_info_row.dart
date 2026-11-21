import 'package:flutter/widgets.dart';

import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';

class TaskInfoRow extends StatelessWidget {
  const TaskInfoRow({
    super.key,
    required this.label,
    required String value,
    this.error = false,
    this.bottomPadding = 14,
  }) : _value = value,
       _child = null;

  const TaskInfoRow.custom({super.key, required this.label, required Widget child, this.bottomPadding = 14})
    : _value = null,
      _child = child,
      error = false;

  final String label;
  final bool error;
  final double bottomPadding;
  final String? _value;
  final Widget? _child;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Padding(
      padding: EdgeInsets.only(bottom: bottomPadding),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 88,
            child: Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                label,
                style: TextStyle(
                  color: palette.textMuted,
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 0.6,
                  height: 1.3,
                ),
              ),
            ),
          ),
          const SizedBox(width: AppDesignTokens.space12),
          Expanded(
            child:
                _child ??
                Text(
                  _value!,
                  style: TextStyle(color: error ? palette.error : palette.textPrimary, fontSize: 12, height: 1.35),
                ),
          ),
        ],
      ),
    );
  }
}
