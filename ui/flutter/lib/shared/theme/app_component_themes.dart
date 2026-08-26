import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

class AppComponentThemes extends StatelessWidget {
  const AppComponentThemes({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = shad.Theme.of(context);
    return shad.ComponentTheme<shad.FocusOutlineTheme>(
      data: shad.FocusOutlineTheme(
        align: 1,
        border: Border.all(color: theme.colorScheme.ring.withValues(alpha: 0.65), width: 1),
      ),
      child: shad.ComponentTheme<shad.ToastTheme>(
        data: const shad.ToastTheme(toastConstraints: BoxConstraints(maxWidth: 420)),
        child: child,
      ),
    );
  }
}
