import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import 'app_design_tokens.dart';
import 'app_palette.dart';

class AppComponentThemes extends StatelessWidget {
  const AppComponentThemes({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = shad.Theme.of(context);
    final palette = AppPalette.of(context);
    return shad.ComponentTheme<shad.FocusOutlineTheme>(
      data: shad.FocusOutlineTheme(
        align: 1,
        border: Border.all(color: theme.colorScheme.ring.withValues(alpha: 0.65), width: 1),
      ),
      child: shad.ComponentTheme<shad.SecondaryButtonTheme>(
        data: shad.SecondaryButtonTheme(
          decoration: (context, states, value) {
            final decoration = switch (value) {
              final BoxDecoration decoration => decoration,
              _ => BoxDecoration(borderRadius: BorderRadius.circular(theme.radiusMd)),
            };
            if (states.contains(WidgetState.disabled)) {
              return decoration.copyWith(color: palette.surfaceSoft);
            }
            if (theme.brightness == Brightness.light) {
              final background = Color.alphaBlend(palette.textPrimary.withValues(alpha: 0.08), palette.surfaceSoft);
              final hoverBackground = Color.alphaBlend(
                palette.textPrimary.withValues(alpha: 0.13),
                palette.surfaceSoft,
              );
              return decoration.copyWith(color: states.contains(WidgetState.hovered) ? hoverBackground : background);
            }
            return value;
          },
          textStyle: (context, states, value) => states.contains(WidgetState.disabled)
              ? value.copyWith(color: palette.textMuted.withValues(alpha: 0.62))
              : value,
          iconTheme: (context, states, value) => states.contains(WidgetState.disabled)
              ? value.copyWith(color: palette.textMuted.withValues(alpha: 0.62))
              : value,
        ),
        child: shad.ComponentTheme<shad.CheckboxTheme>(
          data: shad.CheckboxTheme(
            size: AppDesignTokens.checkboxSize,
            gap: AppDesignTokens.checkboxLabelGap,
            borderColor: theme.brightness == Brightness.dark ? palette.textMuted : palette.border,
          ),
          child: shad.ComponentTheme<shad.ToastTheme>(
            data: const shad.ToastTheme(toastConstraints: BoxConstraints(maxWidth: 420)),
            child: child,
          ),
        ),
      ),
    );
  }
}
