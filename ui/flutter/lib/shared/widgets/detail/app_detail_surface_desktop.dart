import 'package:flutter/material.dart' show Colors;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../theme/app_design_tokens.dart';
import '../../theme/app_palette.dart';

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
                  border: Border(left: BorderSide(color: light ? palette.headerDivider : palette.border)),
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
                      height: AppDesignTokens.contentHeaderHeight,
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 24),
                        child: Align(
                          alignment: AlignmentDirectional.centerStart,
                          child: Text(
                            title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(color: palette.textPrimary, fontSize: 16, fontWeight: FontWeight.w700),
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
