import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../theme/app_palette.dart';

/// The line that explains why Gopeed's own BitTorrent settings are disabled
/// while torrents are handed to rtorrent, with a button to that setting.
class RtorrentHandoffNotice extends StatelessWidget {
  const RtorrentHandoffNotice({super.key, required this.message, required this.actionLabel, required this.onAction});

  final String message;
  final String actionLabel;
  final VoidCallback onAction;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(Icons.info_outline, size: 16, color: palette.textSecondary),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            message,
            key: const ValueKey('rtorrent-handoff-notice-text'),
            style: TextStyle(color: palette.textPrimary, fontSize: 13),
          ),
        ),
        const SizedBox(width: 8),
        shad.SecondaryButton(
          size: shad.ButtonSize.small,
          key: const ValueKey('rtorrent-handoff-notice-action'),
          onPressed: onAction,
          child: Text(actionLabel),
        ),
      ],
    );
  }
}

/// Keeps a setting on screen while it does not apply: greyed out, closed to
/// pointer and keyboard, and announced as disabled.
class DisabledWhen extends StatelessWidget {
  const DisabledWhen({super.key, required this.disabled, required this.child});

  final bool disabled;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (!disabled) {
      return child;
    }
    return Semantics(
      enabled: false,
      child: ExcludeFocus(
        child: AbsorbPointer(child: Opacity(opacity: 0.45, child: child)),
      ),
    );
  }
}
