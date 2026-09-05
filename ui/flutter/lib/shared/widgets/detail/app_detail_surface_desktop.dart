import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../l10n/l10n.dart';
import '../../theme/app_design_tokens.dart';
import '../../theme/app_palette.dart';
import '../app_tooltip.dart';

class AppDetailDrawer extends StatelessWidget {
  const AppDetailDrawer({
    super.key,
    required this.open,
    required this.title,
    required this.onClose,
    required this.child,
    this.drawerKey,
    this.width,
  });

  final bool open;
  final String title;
  final VoidCallback onClose;
  final Widget child;
  final Key? drawerKey;
  final double? width;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final light = shad.Theme.of(context).brightness == Brightness.light;
    final drawerWidth = width ?? AppDesignTokens.taskDetailsDrawerWidth(MediaQuery.sizeOf(context).width);

    return IgnorePointer(
      ignoring: !open,
      child: Stack(
        children: [
          AnimatedOpacity(
            opacity: open ? 1 : 0,
            duration: const Duration(milliseconds: 180),
            child: GestureDetector(
              onTap: onClose,
              child: ColoredBox(color: Colors.black.withValues(alpha: 0.35), child: const SizedBox.expand()),
            ),
          ),
          Align(
            alignment: Alignment.centerRight,
            child: AnimatedSlide(
              duration: const Duration(milliseconds: 260),
              curve: Curves.easeOutCubic,
              offset: open ? Offset.zero : const Offset(1, 0),
              child: Container(
                key: drawerKey,
                width: drawerWidth,
                decoration: BoxDecoration(
                  color: light ? palette.cardBg : palette.bg,
                  border: Border(
                    left: BorderSide(color: light ? palette.headerDivider : palette.border),
                    top: BorderSide(color: palette.headerDivider),
                  ),
                  boxShadow: light
                      ? [
                          BoxShadow(
                            color: Colors.black.withValues(alpha: 0.08),
                            blurRadius: 24,
                            offset: const Offset(-8, 0),
                          ),
                        ]
                      : null,
                ),
                child: Column(
                  children: [
                    SizedBox(
                      key: const ValueKey('app-detail-drawer-header'),
                      height: AppDesignTokens.contentHeaderHeight,
                      child: DecoratedBox(
                        decoration: BoxDecoration(
                          border: Border(bottom: BorderSide(color: palette.headerDivider)),
                        ),
                        child: Padding(
                          padding: const EdgeInsetsDirectional.only(start: 24, end: 16),
                          child: Row(
                            children: [
                              Expanded(
                                child: Text(
                                  title,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(
                                    color: palette.textPrimary,
                                    fontSize: 16,
                                    fontWeight: FontWeight.w700,
                                  ),
                                ),
                              ),
                              const SizedBox(width: 12),
                              AppTooltip(
                                message: context.l10n.close,
                                child: SizedBox.square(
                                  dimension: 28,
                                  child: shad.IconButton.ghost(
                                    key: const ValueKey('app-detail-drawer-close-button'),
                                    size: shad.ButtonSize.xSmall,
                                    onPressed: onClose,
                                    icon: Icon(Icons.close, size: 18, color: palette.textSecondary),
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                    Expanded(child: open ? child : const SizedBox.shrink()),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
