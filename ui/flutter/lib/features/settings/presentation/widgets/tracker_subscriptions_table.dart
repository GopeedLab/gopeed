import 'dart:math' as math;

import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../api/model/downloader_config.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_tooltip.dart';

class TrackerSubscriptionsTable extends StatefulWidget {
  const TrackerSubscriptionsTable({super.key, required this.urls, required this.statuses, required this.onDelete});

  final List<String> urls;
  final Map<String, TrackerSubscriptionStatus> statuses;
  final ValueChanged<int> onDelete;

  @override
  State<TrackerSubscriptionsTable> createState() => _TrackerSubscriptionsTableState();
}

class _TrackerSubscriptionsTableState extends State<TrackerSubscriptionsTable> {
  static const _headerHeight = 40.0;
  static const _rowHeight = 48.0;
  static const _statusWidth = 110.0;
  static const _countWidth = 120.0;
  static const _actionsWidth = 80.0;
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
          final contentWidth = math.max(640.0, constraints.maxWidth);
          final hiddenWidth = contentWidth - constraints.maxWidth;
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
                physics: const ClampingScrollPhysics(),
                child: SizedBox(
                  width: contentWidth,
                  child: Column(
                    children: [
                      Container(
                        height: _headerHeight,
                        decoration: BoxDecoration(
                          color: palette.sideBg,
                          border: Border(bottom: BorderSide(color: palette.border)),
                        ),
                        child: Stack(
                          children: [
                            Positioned.fill(
                              child: Row(
                                children: [
                                  Expanded(child: _header(context.l10n.trackerSubscriptionUrl)),
                                  SizedBox(
                                    width: _statusWidth,
                                    child: _header(context.l10n.trackerSubscriptionUpdateStatus),
                                  ),
                                  SizedBox(
                                    width: _countWidth,
                                    child: _header(context.l10n.trackerSubscriptionTrackerCount),
                                  ),
                                  const SizedBox(width: _actionsWidth),
                                ],
                              ),
                            ),
                            _pinnedActions(
                              hiddenWidth,
                              color: palette.sideBg,
                              child: SizedBox(
                                key: const ValueKey('tracker-subscriptions-actions-header'),
                                child: _header(context.l10n.actions),
                              ),
                            ),
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
                                  itemBuilder: (context, index) => _row(index, hiddenWidth),
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

  // Keep actions inside the same vertical list while compensating for the
  // horizontal viewport offset, so each button stays aligned with its source.
  Widget _pinnedActions(double hiddenWidth, {required Color color, required Widget child}) {
    final palette = AppPalette.of(context);
    return AnimatedBuilder(
      animation: _horizontalController,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: color,
          border: Border(bottom: BorderSide(color: palette.border)),
        ),
        child: Center(child: child),
      ),
      builder: (context, child) => PositionedDirectional(
        top: 0,
        bottom: 0,
        end:
            hiddenWidth - (_horizontalController.hasClients ? _horizontalController.offset.clamp(0.0, hiddenWidth) : 0),
        width: _actionsWidth,
        child: child!,
      ),
    );
  }

  Widget _row(int index, double hiddenWidth) {
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
      child: Stack(
        children: [
          Positioned.fill(
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
                const SizedBox(width: _actionsWidth),
              ],
            ),
          ),
          _pinnedActions(
            hiddenWidth,
            color: palette.cardBg,
            child: AppTooltip(
              message: context.l10n.delete,
              child: SizedBox.square(
                dimension: 44,
                child: GhostButton(
                  key: ValueKey('delete-settings-list-$url'),
                  density: ButtonDensity.icon,
                  onPressed: () => widget.onDelete(index),
                  child: const Icon(Icons.delete_outline, size: 17),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
