import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../api/model/downloader_config.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/widgets/app_loading_button.dart';
import 'settings_list_editor.dart';
import 'tracker_subscriptions_schedule.dart';
import 'tracker_subscriptions_table.dart';

class TrackerSubscriptionsControl extends StatefulWidget {
  const TrackerSubscriptionsControl({
    super.key,
    required this.urls,
    required this.statuses,
    required this.autoUpdate,
    required this.lastUpdated,
    required this.onChanged,
    required this.onAutoUpdateChanged,
    required this.onUpdate,
  });

  final List<String> urls;
  final Map<String, TrackerSubscriptionStatus> statuses;
  final bool autoUpdate;
  final DateTime? lastUpdated;
  final ValueChanged<List<String>> onChanged;
  final ValueChanged<bool> onAutoUpdateChanged;
  final Future<void> Function() onUpdate;

  @override
  State<TrackerSubscriptionsControl> createState() => _TrackerSubscriptionsControlState();
}

class _TrackerSubscriptionsControlState extends State<TrackerSubscriptionsControl> {
  bool _updating = false;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: double.infinity,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TrackerSubscriptionsSchedule(
            lastUpdated: widget.lastUpdated,
            autoUpdate: widget.autoUpdate,
            onAutoUpdateChanged: widget.onAutoUpdateChanged,
          ),
          const SizedBox(height: AppDesignTokens.space12),
          TrackerSubscriptionsTable(urls: widget.urls, statuses: widget.statuses, onDelete: _delete),
          const SizedBox(height: AppDesignTokens.space12),
          Wrap(
            key: const ValueKey('tracker-subscriptions-actions'),
            alignment: WrapAlignment.end,
            crossAxisAlignment: WrapCrossAlignment.center,
            spacing: AppDesignTokens.space8,
            runSpacing: AppDesignTokens.space8,
            children: [
              AppLoadingButton(
                key: const ValueKey('tracker-update-button'),
                onPressed: _update,
                loading: _updating,
                icon: const Icon(Icons.refresh, size: 17),
                child: Text(context.l10n.newVersionUpdate),
              ),
              SecondaryButton(
                key: const ValueKey('add-tracker-subscription'),
                onPressed: _add,
                leading: const Icon(Icons.add, size: 17),
                child: Text(context.l10n.add),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Future<void> _add() async {
    final value = await showTextSettingDialog(
      context,
      title: context.l10n.addTrackerSubscription,
      fieldLabel: context.l10n.trackerSubscriptionUrl,
      requireHttpUrl: true,
    );
    if (!mounted || value == null) return;
    final next = [...widget.urls, value];
    widget.onChanged(next.toSet().toList());
  }

  Future<void> _delete(int index) async {
    final url = widget.urls[index];
    if (!await showDeleteSettingsEntryDialog(context, url) || !mounted) return;
    widget.onChanged(List<String>.from(widget.urls)..remove(url));
  }

  Future<void> _update() async {
    setState(() => _updating = true);
    try {
      await widget.onUpdate();
    } finally {
      if (mounted) setState(() => _updating = false);
    }
  }
}
