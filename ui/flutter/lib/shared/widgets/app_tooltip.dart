import 'package:flutter/material.dart' show Tooltip, TooltipState;
import 'package:flutter/widgets.dart';

/// The single product tooltip primitive.
///
/// Flutter's tooltip is used intentionally because the shadcn tooltip does not
/// currently inherit Gopeed's mixed Shadcn/Material theme consistently.
class AppTooltip extends StatefulWidget {
  const AppTooltip({super.key, required this.message, required this.child, this.focusable = false});

  final String message;
  final Widget child;
  final bool focusable;

  @override
  State<AppTooltip> createState() => _AppTooltipState();
}

class _AppTooltipState extends State<AppTooltip> {
  final _tooltipKey = GlobalKey<TooltipState>();

  @override
  Widget build(BuildContext context) {
    final tooltip = Tooltip(key: _tooltipKey, message: widget.message, child: widget.child);
    if (!widget.focusable) return tooltip;
    return Focus(
      onFocusChange: (focused) {
        if (focused) {
          _tooltipKey.currentState?.ensureTooltipVisible();
        } else {
          Tooltip.dismissAllToolTips();
        }
      },
      child: tooltip,
    );
  }
}
