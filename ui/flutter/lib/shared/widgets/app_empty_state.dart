import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';
import 'app_tooltip.dart';
import 'app_empty_illustration.dart';

export 'app_empty_illustration.dart' show AppEmptyIllustrationType;

/// Shared empty/error surface used by task and extension views.
class AppEmptyState extends StatelessWidget {
  const AppEmptyState({
    super.key,
    required this.message,
    required this.illustration,
    this.action,
    this.illustrationKey,
  });

  final String message;
  final AppEmptyIllustrationType illustration;
  final Widget? action;
  final Key? illustrationKey;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Center(
      child: SingleChildScrollView(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 360),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                AppEmptyIllustration(key: illustrationKey, type: illustration),
                const SizedBox(height: 18),
                Text(
                  message,
                  textAlign: TextAlign.center,
                  style: TextStyle(color: palette.textPrimary, fontSize: 17, fontWeight: FontWeight.w700),
                ),
                if (action != null) ...[const SizedBox(height: AppDesignTokens.space24), action!],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// A small outlined action icon beside a muted label, matching task guidance.
class AppEmptyStateAction extends StatelessWidget {
  const AppEmptyStateAction({
    super.key,
    required this.icon,
    required this.label,
    required this.onPressed,
    this.buttonKey,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  final Key? buttonKey;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        AppTooltip(
          message: label,
          child: SizedBox.square(
            dimension: 30,
            child: shad.IconButton.outline(
              key: buttonKey,
              density: shad.ButtonDensity.compact,
              onPressed: onPressed,
              icon: Icon(icon, size: 16),
            ),
          ),
        ),
        const SizedBox(width: 8),
        Flexible(
          child: Text(label, style: TextStyle(color: AppPalette.of(context).textMuted, fontSize: 13)),
        ),
      ],
    );
  }
}
