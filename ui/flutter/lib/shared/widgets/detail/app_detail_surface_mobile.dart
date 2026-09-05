import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../l10n/l10n.dart';
import '../../theme/app_palette.dart';
import '../app_tooltip.dart';

class AppDetailPage extends StatelessWidget {
  const AppDetailPage({
    super.key,
    required this.title,
    required this.onBack,
    required this.child,
    this.backgroundColor,
  });

  final String title;
  final VoidCallback onBack;
  final Widget child;
  final Color? backgroundColor;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final background = backgroundColor ?? palette.bg;

    return shad.Scaffold(
      backgroundColor: background,
      child: ColoredBox(
        color: background,
        child: Column(
          children: [
            SafeArea(
              bottom: false,
              child: Container(
                height: 52,
                padding: const EdgeInsets.symmetric(horizontal: 8),
                decoration: BoxDecoration(
                  border: Border(bottom: BorderSide(color: palette.border)),
                ),
                child: Row(
                  children: [
                    SizedBox(
                      width: 44,
                      child: AppTooltip(
                        message: context.l10n.back,
                        child: shad.IconButton.ghost(
                          key: const ValueKey('app-detail-back-button'),
                          onPressed: onBack,
                          icon: Icon(Icons.arrow_back, size: 20, color: palette.textPrimary),
                        ),
                      ),
                    ),
                    Expanded(
                      child: Text(
                        title,
                        textAlign: TextAlign.center,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(color: palette.textPrimary, fontSize: 17, fontWeight: FontWeight.w700),
                      ),
                    ),
                    const SizedBox(width: 44),
                  ],
                ),
              ),
            ),
            Expanded(child: child),
          ],
        ),
      ),
    );
  }
}
