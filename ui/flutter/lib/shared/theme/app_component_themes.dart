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
            if (!states.contains(WidgetState.disabled)) return value;
            return switch (value) {
              final BoxDecoration decoration => decoration.copyWith(color: palette.surfaceSoft),
              _ => BoxDecoration(color: palette.surfaceSoft, borderRadius: BorderRadius.circular(theme.radiusMd)),
            };
          },
          textStyle: (context, states, value) => states.contains(WidgetState.disabled)
              ? value.copyWith(color: palette.textMuted.withValues(alpha: 0.62))
              : value,
          iconTheme: (context, states, value) => states.contains(WidgetState.disabled)
              ? value.copyWith(color: palette.textMuted.withValues(alpha: 0.62))
              : value,
        ),
        child: shad.ComponentTheme<shad.CheckboxTheme>(
          data: const shad.CheckboxTheme(size: AppDesignTokens.checkboxSize, gap: AppDesignTokens.checkboxLabelGap),
          child: shad.ComponentTheme<shad.ToastTheme>(
            data: const shad.ToastTheme(toastConstraints: BoxConstraints(maxWidth: 420)),
            child: child,
          ),
        ),
      ),
    );
  }
}
