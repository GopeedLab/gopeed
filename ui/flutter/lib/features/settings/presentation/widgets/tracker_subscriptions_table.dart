import 'dart:math' as math;

import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../api/model/downloader_config.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_tooltip.dart';

class TrackerSubscriptionsTable extends StatefulWidget {
  const TrackerSubscriptionsTable({
    super.key,
    required this.urls,
    required this.statuses,
    required this.onEdit,
    required this.onDelete,
  });

  final List<String> urls;
  final Map<String, TrackerSubscriptionStatus> statuses;
  final ValueChanged<int> onEdit;
  final ValueChanged<int> onDelete;

  @override
  State<TrackerSubscriptionsTable> createState() => _TrackerSubscriptionsTableState();
}

class _TrackerSubscriptionsTableState extends State<TrackerSubscriptionsTable> {
  static const _headerHeight = 40.0;
  static const _rowHeight = 48.0;
  static const _statusWidth = 110.0;
  static const _countWidth = 120.0;
  static const _actionsWidth = 96.0;
  final _horizontalController = ScrollController();
  final _verticalController = ScrollController();

  @override
  void dispose() {
    _horizontalController.dispose();
    _verticalController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final height = _headerHeight + _rowHeight * widget.urls.length.clamp(1, 5) + 12;
    return Container(
      key: const ValueKey('tracker-subscriptions-table'),
      height: height,
      clipBehavior: Clip.hardEdge,
      decoration: BoxDecoration(
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          return Scrollbar(
            key: const ValueKey('tracker-subscriptions-horizontal-scrollbar'),
            controller: _horizontalController,
            interactive: true,
            thickness: 7,
            scrollbarOrientation: ScrollbarOrientation.bottom,
            notificationPredicate: (notification) => notification.metrics.axis == Axis.horizontal,
            child: Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: SingleChildScrollView(
                controller: _horizontalController,
                scrollDirection: Axis.horizontal,
                child: SizedBox(
                  width: math.max(640, constraints.maxWidth),
                  child: Column(
                    children: [
                      Container(
                        height: _headerHeight,
                        decoration: BoxDecoration(
                          color: palette.sideBg,
                          border: Border(bottom: BorderSide(color: palette.border)),
                        ),
                        child: Row(
                          children: [
                            Expanded(child: _header(context.l10n.trackerSubscriptionUrl)),
                            SizedBox(width: _statusWidth, child: _header(context.l10n.trackerSubscriptionUpdateStatus)),
                            SizedBox(width: _countWidth, child: _header(context.l10n.trackerSubscriptionTrackerCount)),
                            const SizedBox(width: _actionsWidth),
                          ],
                        ),
                      ),
                      Expanded(
                        child: widget.urls.isEmpty
                            ? Center(
                                child: Text('—', style: TextStyle(color: palette.textMuted)),
                              )
                            : Scrollbar(
                                controller: _verticalController,
                                thumbVisibility: true,
                                notificationPredicate: (notification) => notification.metrics.axis == Axis.vertical,
                                child: ListView.builder(
                                  key: const ValueKey('tracker-subscriptions-rows'),
                                  controller: _verticalController,
                                  padding: EdgeInsets.zero,
                                  itemExtent: _rowHeight,
                                  itemCount: widget.urls.length,
                                  itemBuilder: (context, index) => _row(index),
                                ),
                              ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  Widget _header(String label) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 10),
      child: Text(
        label,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(color: AppPalette.of(context).textSecondary, fontSize: 11, fontWeight: FontWeight.w700),
      ),
    );
  }

  Widget _row(int index) {
    final palette = AppPalette.of(context);
    final url = widget.urls[index];
    final status = widget.statuses[url];
    final result = status == null
        ? context.l10n.trackerSubscriptionNotUpdated
        : status.success
        ? context.l10n.trackerSubscriptionFetchSuccess
        : context.l10n.trackerSubscriptionFetchFailed;
    return DecoratedBox(
      key: ValueKey('tracker-subscription-$url'),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          Expanded(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 10),
              child: AppTooltip(
                message: url,
                child: Text(
                  url,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(color: palette.textPrimary, fontSize: 12),
                ),
              ),
            ),
          ),
          SizedBox(
            width: _statusWidth,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 10),
              child: Text(
                result,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  color: status == null
                      ? palette.textMuted
                      : status.success
                      ? palette.success
                      : palette.error,
                  fontSize: 11,
                ),
              ),
            ),
          ),
          SizedBox(
            width: _countWidth,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 10),
              child: Text(
                status?.trackerCount.toString() ?? '—',
                style: TextStyle(color: palette.textSecondary, fontSize: 11),
              ),
            ),
          ),
          SizedBox(
            width: _actionsWidth,
            child: Row(
              children: [
                AppTooltip(
                  message: context.l10n.edit,
                  child: GhostButton(
                    key: ValueKey('edit-settings-list-$url'),
                    density: ButtonDensity.icon,
                    onPressed: () => widget.onEdit(index),
                    child: const Icon(Icons.edit_outlined, size: 17),
                  ),
                ),
                AppTooltip(
                  message: context.l10n.delete,
                  child: GhostButton(
                    key: ValueKey('delete-settings-list-$url'),
                    density: ButtonDensity.icon,
                    onPressed: () => widget.onDelete(index),
                    child: const Icon(Icons.delete_outline, size: 17),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
