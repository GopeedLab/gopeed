import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';

/// A quiet disclosure control with the same keyboard behavior as other buttons.
class CreateTaskAdvancedToggle extends StatelessWidget {
  const CreateTaskAdvancedToggle({super.key, required this.expanded, required this.onPressed});

  final bool expanded;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final reduceMotion = MediaQuery.disableAnimationsOf(context);
    return Semantics(
      expanded: expanded,
      child: ButtonStyleOverride.inherit(
        padding: (context, states, value) => const EdgeInsets.symmetric(vertical: AppDesignTokens.space12),
        child: GhostButton(
          onPressed: onPressed,
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 20),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Flexible(
                  child: Text(
                    context.l10n.advanced,
                    style: TextStyle(color: palette.textPrimary, fontSize: 14, fontWeight: FontWeight.w600),
                  ),
                ),
                const SizedBox(width: AppDesignTokens.space8),
                AnimatedRotation(
                  turns: expanded ? 0.5 : 0,
                  duration: reduceMotion ? Duration.zero : const Duration(milliseconds: 180),
                  curve: Curves.easeOutCubic,
                  child: Icon(Icons.keyboard_arrow_down, size: 16, color: palette.textSecondary),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
