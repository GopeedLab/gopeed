import 'package:flutter/gestures.dart' show PointerDeviceKind;
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
    this.scrollValue = false,
  }) : _value = value,
       _child = null;

  const TaskInfoRow.custom({super.key, required this.label, required Widget child, this.bottomPadding = 14})
    : _value = null,
      _child = child,
      scrollValue = false,
      error = false;

  final String label;
  final bool error;
  final double bottomPadding;
  final bool scrollValue;
  final String? _value;
  final Widget? _child;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final value =
        _child ??
        Text(
          _value!,
          maxLines: scrollValue ? 1 : null,
          style: TextStyle(color: error ? palette.error : palette.textPrimary, fontSize: 12, height: 1.35),
        );
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
            child: scrollValue
                ? ScrollConfiguration(
                    behavior: ScrollConfiguration.of(context).copyWith(
                      dragDevices: {...ScrollConfiguration.of(context).dragDevices, PointerDeviceKind.mouse},
                      scrollbars: false,
                    ),
                    child: SingleChildScrollView(scrollDirection: Axis.horizontal, child: value),
                  )
                : value,
          ),
        ],
      ),
    );
  }
}
