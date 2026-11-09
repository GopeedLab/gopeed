import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../shared/theme/app_design_tokens.dart';
import '../../shared/theme/app_palette.dart';
import '../../core/utils/breakpoints.dart';

/// A routed modal keeps its underlying page mounted and participates in Back.
class AppWindowPage<T> extends Page<T> {
  const AppWindowPage({super.key, required this.child, this.preferredSize = AppDesignTokens.childWindowSize});

  final Widget child;
  final Size preferredSize;

  @override
  Route<T> createRoute(BuildContext context) => DialogRoute<T>(
    context: context,
    settings: this,
    barrierDismissible: false,
    barrierColor: Theme.of(context).colorScheme.background.withValues(alpha: 0.6),
    traversalEdgeBehavior: TraversalEdgeBehavior.closedLoop,
    alignment: Alignment.center,
    fullScreen: true,
    transitionBuilder: (context, animation, secondaryAnimation, child) =>
        FadeTransition(opacity: animation, child: child),
    builder: (context) => Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final page = ModalRoute.settingsOf(context)! as AppWindowPage<T>;
          final palette = AppPalette.of(context);
          return Padding(
            padding: EdgeInsets.all(
              constraints.maxWidth < Breakpoints.mobile ? AppDesignTokens.space12 : AppDesignTokens.space32,
            ),
            child: Center(
              child: ConstrainedBox(
                constraints: BoxConstraints(maxWidth: page.preferredSize.width, maxHeight: page.preferredSize.height),
                child: ModalContainer(
                  key: const ValueKey('app-window-modal'),
                  padding: EdgeInsets.zero,
                  filled: true,
                  fillColor: palette.sideBg,
                  borderColor: palette.border,
                  borderRadius: BorderRadius.circular(AppDesignTokens.windowRadius),
                  clipBehavior: Clip.antiAlias,
                  child: MediaQuery.removeViewInsets(
                    context: context,
                    removeBottom: true,
                    child: MediaQuery.removePadding(
                      context: context,
                      removeTop: true,
                      removeBottom: true,
                      child: page.child,
                    ),
                  ),
                ),
              ),
            ),
          );
        },
      ),
    ),
  );
}
