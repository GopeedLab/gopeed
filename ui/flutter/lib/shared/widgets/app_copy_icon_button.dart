import 'package:flutter/material.dart' show Icons;
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../l10n/l10n.dart';
import 'app_tooltip.dart';

class AppCopyIconButton extends StatefulWidget {
  const AppCopyIconButton({
    super.key,
    required this.text,
    this.enabled = true,
    this.dimension = 28,
    this.iconSize = 16,
  });

  final String text;
  final bool enabled;
  final double dimension;
  final double iconSize;

  @override
  State<AppCopyIconButton> createState() => _AppCopyIconButtonState();
}

class _AppCopyIconButtonState extends State<AppCopyIconButton> {
  bool _copied = false;

  @override
  void didUpdateWidget(covariant AppCopyIconButton oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.text != oldWidget.text || (!widget.enabled && oldWidget.enabled)) {
      _copied = false;
    }
  }

  @override
  Widget build(BuildContext context) {
    return AppTooltip(
      message: _copied ? context.l10n.copied : context.l10n.copy,
      child: SizedBox.square(
        dimension: widget.dimension,
        child: shad.IconButton.ghost(
          size: shad.ButtonSize.xSmall,
          onPressed: widget.enabled ? _copy : null,
          icon: Icon(_copied ? Icons.check : Icons.copy_outlined, size: widget.iconSize),
        ),
      ),
    );
  }

  Future<void> _copy() async {
    await Clipboard.setData(ClipboardData(text: widget.text));
    if (mounted) {
      setState(() => _copied = true);
    }
  }
}
