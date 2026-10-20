import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_loading_button.dart';

class TrackerSubscriptionsFooter extends StatelessWidget {
  const TrackerSubscriptionsFooter({
    super.key,
    required this.lastUpdated,
    required this.autoUpdate,
    required this.updating,
    required this.onAutoUpdateChanged,
    required this.onUpdate,
  });

  final DateTime? lastUpdated;
  final bool autoUpdate;
  final bool updating;
  final ValueChanged<bool> onAutoUpdateChanged;
  final VoidCallback onUpdate;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Wrap(
      key: const ValueKey('tracker-subscriptions-footer'),
      alignment: WrapAlignment.end,
      crossAxisAlignment: WrapCrossAlignment.center,
      spacing: AppDesignTokens.space16,
      runSpacing: AppDesignTokens.space8,
      children: [
        if (lastUpdated case final time?)
          Text(
            key: const ValueKey('tracker-last-updated'),
            context.l10n.lastUpdated(_formatDateTime(time)),
            textAlign: TextAlign.end,
            style: TextStyle(color: palette.textMuted, fontSize: 11),
          ),
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Flexible(
              child: Row(
                key: const ValueKey('tracker-auto-update-control'),
                mainAxisSize: MainAxisSize.min,
                children: [
                  Flexible(
                    child: Text(
                      context.l10n.updateEveryDay,
                      style: TextStyle(color: palette.textSecondary, fontSize: 12),
                    ),
                  ),
                  const SizedBox(width: AppDesignTokens.space8),
                  Switch(value: autoUpdate, onChanged: onAutoUpdateChanged),
                ],
              ),
            ),
            const SizedBox(width: AppDesignTokens.space16),
            AppLoadingButton(
              key: const ValueKey('tracker-update-button'),
              onPressed: onUpdate,
              loading: updating,
              icon: const Icon(Icons.refresh, size: 17),
              child: Text(context.l10n.newVersionUpdate),
            ),
          ],
        ),
      ],
    );
  }

  String _formatDateTime(DateTime time) {
    final value = time.toLocal();
    String two(int number) => number.toString().padLeft(2, '0');
    return '${value.year}-${two(value.month)}-${two(value.day)} ${two(value.hour)}:${two(value.minute)}:${two(value.second)}';
  }
}
