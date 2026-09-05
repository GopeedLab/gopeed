import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';

class AppPrimaryButton extends StatelessWidget {
  const AppPrimaryButton({
    super.key,
    required this.child,
    this.onPressed,
    this.leading,
    this.trailing,
    this.alignment,
    this.size = shad.ButtonSize.normal,
    this.density = shad.ButtonDensity.normal,
    this.shape = shad.ButtonShape.rectangle,
    this.disableTransition = false,
  });

  final Widget child;
  final VoidCallback? onPressed;
  final Widget? leading;
  final Widget? trailing;
  final AlignmentGeometry? alignment;
  final shad.ButtonSize size;
  final shad.ButtonDensity density;
  final shad.ButtonShape shape;
  final bool disableTransition;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);

    return shad.ButtonStyleOverride(
      decoration: (context, states, value) {
        return BoxDecoration(
          color: _resolveBackground(palette, states),
          borderRadius: shape == shad.ButtonShape.circle ? null : BorderRadius.circular(AppDesignTokens.controlRadius),
          shape: shape == shad.ButtonShape.circle ? BoxShape.circle : BoxShape.rectangle,
        );
      },
      textStyle: (context, states, value) {
        return value.copyWith(color: _resolveForeground(palette, states));
      },
      iconTheme: (context, states, value) {
        return value.copyWith(color: _resolveForeground(palette, states));
      },
      child: shad.PrimaryButton(
        onPressed: onPressed,
        leading: leading,
        trailing: trailing,
        alignment: alignment,
        size: size,
        density: density,
        shape: shape,
        disableTransition: disableTransition,
        child: child,
      ),
    );
  }

  Color _resolveBackground(AppPalette palette, Set<WidgetState> states) {
    if (states.contains(WidgetState.disabled)) {
      return palette.primaryActionBg.withValues(alpha: 0.38);
    }
    if (states.contains(WidgetState.pressed)) {
      return Color.alphaBlend(palette.primaryActionForeground.withValues(alpha: 0.14), palette.primaryActionBg);
    }
    if (states.contains(WidgetState.hovered)) {
      return palette.primaryActionHoverBg;
    }
    return palette.primaryActionBg;
  }

  Color _resolveForeground(AppPalette palette, Set<WidgetState> states) {
    if (states.contains(WidgetState.disabled)) {
      return palette.primaryActionForeground.withValues(alpha: 0.55);
    }
    return palette.primaryActionForeground;
  }
}
