import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';

class TrackerSubscriptionsSchedule extends StatelessWidget {
  const TrackerSubscriptionsSchedule({
    super.key,
    required this.lastUpdated,
    required this.autoUpdate,
    required this.onAutoUpdateChanged,
  });

  final DateTime? lastUpdated;
  final bool autoUpdate;
  final ValueChanged<bool> onAutoUpdateChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Column(
      key: const ValueKey('tracker-subscriptions-schedule'),
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Row(
          key: const ValueKey('tracker-auto-update-control'),
          mainAxisSize: MainAxisSize.min,
          children: [
            Flexible(
              child: Text(context.l10n.updateEveryDay, style: TextStyle(color: palette.textSecondary, fontSize: 12)),
            ),
            const SizedBox(width: AppDesignTokens.space8),
            Switch(value: autoUpdate, onChanged: onAutoUpdateChanged),
          ],
        ),
        if (lastUpdated case final time?) ...[
          const SizedBox(height: AppDesignTokens.space4),
          Text(
            key: const ValueKey('tracker-last-updated'),
            context.l10n.lastUpdated(_formatDateTime(time)),
            textAlign: TextAlign.end,
            style: TextStyle(color: palette.textMuted, fontSize: 11),
          ),
        ],
      ],
    );
  }

  String _formatDateTime(DateTime time) {
    final value = time.toLocal();
    String two(int number) => number.toString().padLeft(2, '0');
    return '${value.year}-${two(value.month)}-${two(value.day)} ${two(value.hour)}:${two(value.minute)}:${two(value.second)}';
  }
}
