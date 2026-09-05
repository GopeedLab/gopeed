import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';
import 'app_primary_button.dart';

enum AppLoadingButtonVariant { secondary, primary, brand }

class AppLoadingButton extends StatelessWidget {
  const AppLoadingButton({
    super.key,
    required this.onPressed,
    required this.loading,
    required this.child,
    this.icon,
    this.alignment,
    this.variant = AppLoadingButtonVariant.secondary,
  });

  final VoidCallback? onPressed;
  final bool loading;
  final Widget child;
  final Widget? icon;
  final AlignmentGeometry? alignment;
  final AppLoadingButtonVariant variant;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final replaceChild = loading && icon == null;
    final leading = loading && icon != null ? _LoadingIcon(variant: variant) : icon;
    final effectiveChild = replaceChild ? _LoadingIcon(variant: variant) : child;
    final content = leading == null
        ? effectiveChild
        : Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              leading,
              const SizedBox(width: AppDesignTokens.space8),
              effectiveChild,
            ],
          );
    final effectiveOnPressed = loading ? null : onPressed;

    return switch (variant) {
      AppLoadingButtonVariant.secondary => shad.SecondaryButton(
        onPressed: effectiveOnPressed,
        alignment: alignment,
        child: content,
      ),
      AppLoadingButtonVariant.primary => AppPrimaryButton(
        onPressed: effectiveOnPressed,
        alignment: alignment,
        child: content,
      ),
      AppLoadingButtonVariant.brand => shad.ButtonStyleOverride(
        decoration: (context, states, value) {
          final disabled = states.contains(WidgetState.disabled);
          final color = loading
              ? palette.brand
              : disabled
              ? palette.brand.withValues(alpha: 0.38)
              : states.contains(WidgetState.pressed)
              ? Color.alphaBlend(palette.brandForeground.withValues(alpha: 0.12), palette.brand)
              : states.contains(WidgetState.hovered)
              ? Color.alphaBlend(palette.brandForeground.withValues(alpha: 0.08), palette.brand)
              : palette.brand;
          return BoxDecoration(color: color, borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius));
        },
        textStyle: (context, states, value) => value.copyWith(
          color: states.contains(WidgetState.disabled) && !loading
              ? palette.brandForeground.withValues(alpha: 0.55)
              : palette.brandForeground,
        ),
        iconTheme: (context, states, value) => value.copyWith(
          color: states.contains(WidgetState.disabled) && !loading
              ? palette.brandForeground.withValues(alpha: 0.55)
              : palette.brandForeground,
        ),
        child: shad.SecondaryButton(onPressed: effectiveOnPressed, alignment: alignment, child: content),
      ),
    };
  }
}

class _LoadingIcon extends StatelessWidget {
  const _LoadingIcon({required this.variant});

  final AppLoadingButtonVariant variant;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final foreground = switch (variant) {
      AppLoadingButtonVariant.secondary => palette.textMuted,
      AppLoadingButtonVariant.primary => palette.primaryActionForeground,
      AppLoadingButtonVariant.brand => palette.brandForeground,
    };
    final background = switch (variant) {
      AppLoadingButtonVariant.secondary => palette.border,
      AppLoadingButtonVariant.primary || AppLoadingButtonVariant.brand => foreground.withValues(alpha: 0.24),
    };
    return Center(
      child: SizedBox.square(
        dimension: 14,
        child: shad.CircularProgressIndicator(size: 14, color: foreground, backgroundColor: background),
      ),
    );
  }
}
