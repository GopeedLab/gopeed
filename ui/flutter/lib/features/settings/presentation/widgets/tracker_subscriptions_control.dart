import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../api/model/downloader_config.dart';
import '../../../../core/utils/breakpoints.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_loading_button.dart';
import 'settings_list_editor.dart';

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
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    final palette = AppPalette.of(context);
    return SizedBox(
      width: desktop ? AppDesignTokens.settingsFormControlWidth : double.infinity,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SettingsListEditor(
            entries: [
              for (final url in widget.urls)
                SettingsListEntry(id: url, title: url, subtitle: _statusText(context, widget.statuses[url])),
            ],
            addLabel: context.l10n.addTrackerSubscription,
            addButtonKey: const ValueKey('add-tracker-subscription'),
            onAdd: () => _edit(),
            onEdit: _edit,
            onDelete: _delete,
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              AppLoadingButton(
                key: const ValueKey('tracker-update-button'),
                onPressed: _update,
                loading: _updating,
                icon: const Icon(Icons.refresh, size: 17),
                child: Text(context.l10n.newVersionUpdate),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  context.l10n.updateEveryDay,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  textAlign: TextAlign.end,
                  style: TextStyle(color: palette.textSecondary, fontSize: 12),
                ),
              ),
              const SizedBox(width: 8),
              Switch(value: widget.autoUpdate, onChanged: widget.onAutoUpdateChanged),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            widget.lastUpdated == null
                ? context.l10n.neverUpdated
                : context.l10n.lastUpdated(_formatDateTime(widget.lastUpdated!)),
            style: TextStyle(color: palette.textMuted, fontSize: 11),
          ),
        ],
      ),
    );
  }

  String _statusText(BuildContext context, TrackerSubscriptionStatus? status) {
    if (status == null) return context.l10n.neverUpdated;
    final result = status.success
        ? context.l10n.trackerSubscriptionFetchSuccess(status.trackerCount)
        : context.l10n.trackerSubscriptionFetchFailed;
    return '$result · ${_formatDateTime(status.lastFetchTime)}';
  }

  Future<void> _edit([int? index]) async {
    final original = index == null ? null : widget.urls[index];
    final value = await showTextSettingDialog(
      context,
      title: original == null ? context.l10n.addTrackerSubscription : context.l10n.editTrackerSubscription,
      fieldLabel: context.l10n.trackerSubscriptionUrl,
      initialValue: original ?? '',
      requireHttpUrl: true,
    );
    if (!mounted || value == null) return;
    final next = List<String>.from(widget.urls);
    if (original == null) {
      next.add(value);
    } else {
      final currentIndex = next.indexOf(original);
      if (currentIndex == -1) return;
      next[currentIndex] = value;
    }
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

  String _formatDateTime(DateTime time) {
    final value = time.toLocal();
    String two(int number) => number.toString().padLeft(2, '0');
    return '${value.year}-${two(value.month)}-${two(value.day)} ${two(value.hour)}:${two(value.minute)}:${two(value.second)}';
  }
}
