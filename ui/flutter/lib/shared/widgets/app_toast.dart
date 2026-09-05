import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../theme/app_palette.dart';

enum AppToastType { info, success, error }

/// Shows the application's shared toast presentation.
///
/// Toast width is controlled by the loose global [shad.ToastTheme] constraint:
/// short messages stay compact while long messages wrap before becoming wider
/// than the configured maximum. Errors and normal notices use the same layout.
void showAppToast(BuildContext context, String message, {AppToastType type = AppToastType.info}) {
  final palette = AppPalette.of(context);
  final borderColor = switch (type) {
    AppToastType.error => palette.error.withValues(alpha: 0.55),
    AppToastType.success || AppToastType.info => palette.border,
  };
  shad.showToast(
    context: context,
    location: shad.ToastLocation.topCenter,
    builder: (context, overlay) => Align(
      alignment: Alignment.topCenter,
      widthFactor: 1,
      heightFactor: 1,
      child: Container(
        key: const ValueKey('app-toast-content'),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        decoration: BoxDecoration(
          color: palette.cardBg,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: borderColor),
        ),
        child: Text(message, style: TextStyle(color: palette.textPrimary, fontSize: 13, height: 1.2)),
      ),
    ),
  );
}
