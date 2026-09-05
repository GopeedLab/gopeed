import 'dart:math' as math;

import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../../api/model/task_stats.dart';
import '../../../../../core/utils/byte_size_formatter.dart';
import '../../../../../shared/theme/app_design_tokens.dart';
import '../../../../../shared/theme/app_palette.dart';
import '../../../../../shared/widgets/app_tooltip.dart';
import '../../../../../l10n/l10n.dart';
import '../../../application/task_stats_provider.dart';
import '../../../domain/task_record.dart';
import '../../task_record_localizations.dart';
import '../task_progress_bar.dart';
import 'peer_table.dart';
import 'piece_map.dart';

class TaskStatisticsTab extends ConsumerWidget {
  const TaskStatisticsTab({super.key, required this.task, required this.mobile});

  final TaskRecord task;
  final bool mobile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final statsAsync = ref.watch(taskStatsProvider((taskId: task.id, protocol: task.protocol)));
    final stats = statsAsync.value;
    final horizontalPadding = mobile
        ? AppDesignTokens.taskDetailsMobilePadding
        : AppDesignTokens.taskDetailsDesktopPadding;

    if (stats == null && statsAsync.isLoading) {
      return Padding(padding: EdgeInsets.all(horizontalPadding), child: const _StatisticsSkeleton());
    }
    if (stats == null) {
      return const _StatisticsEmptyState();
    }

    return SingleChildScrollView(
      key: PageStorageKey<String>('task-statistics-${task.id}'),
      padding: EdgeInsets.fromLTRB(horizontalPadding, horizontalPadding, horizontalPadding, 32),
      child: switch (stats) {
        HttpTaskStats() => _HttpStatistics(stats: stats, taskStatus: task.status),
        BtTaskStats() => _BtStatistics(stats: stats, task: task),
        Ed2kTaskStats() => _Ed2kStatistics(stats: stats),
      },
    );
  }
}

class _HttpStatistics extends StatelessWidget {
  const _HttpStatistics({required this.stats, required this.taskStatus});

  final HttpTaskStats stats;
  final TaskStatus taskStatus;

  @override
  Widget build(BuildContext context) {
    final completed = stats.connections.where((connection) => connection.completed).length;
    final failed = stats.connections.where((connection) => connection.failed).length;
    final downloading = stats.connections.length - completed - failed;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionTitle(context.l10n.connectionStatus, trailing: context.l10n.connectionCount(stats.connections.length)),
        const SizedBox(height: 12),
        _SummaryGrid(
          items: [
            _SummaryItem(context.l10n.downloading, math.max(0, downloading).toString()),
            _SummaryItem(context.l10n.completed, completed.toString()),
            _SummaryItem(context.l10n.failed, failed.toString()),
          ],
        ),
        const SizedBox(height: 18),
        HttpConnectionLanes(connections: stats.connections, taskDownloading: taskStatus == TaskStatus.downloading),
      ],
    );
  }
}

class HttpConnectionLanes extends StatelessWidget {
  const HttpConnectionLanes({super.key, required this.connections, required this.taskDownloading});

  final List<HttpConnectionStats> connections;
  final bool taskDownloading;

  @override
  Widget build(BuildContext context) {
    if (connections.isEmpty) {
      final palette = AppPalette.of(context);
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 48),
        child: Center(
          child: Text(context.l10n.noPeers, style: TextStyle(color: palette.textMuted, fontSize: 13)),
        ),
      );
    }
    return Column(
      children: [
        for (var index = 0; index < connections.length; index++)
          _ConnectionLane(index: index, connection: connections[index], taskDownloading: taskDownloading),
      ],
    );
  }
}

class _ConnectionLane extends StatelessWidget {
  const _ConnectionLane({required this.index, required this.connection, required this.taskDownloading});

  final int index;
  final HttpConnectionStats connection;
  final bool taskDownloading;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final retrying = !connection.completed && !connection.failed && connection.retryTimes > 0;
    final downloading = taskDownloading && !retrying && !connection.completed && !connection.failed;
    final hasTotal = connection.total > 0;
    final progress = hasTotal ? (connection.downloaded / connection.total).clamp(0.0, 1.0) : null;
    final status = connection.failed
        ? context.l10n.failed
        : connection.completed
        ? context.l10n.completed
        : retrying
        ? context.l10n.retrying
        : context.l10n.downloading;
    final statusColor = connection.failed
        ? palette.error
        : connection.completed
        ? palette.success
        : palette.brand;
    return LayoutBuilder(
      builder: (context, constraints) {
        final narrow = constraints.maxWidth < 330;
        final amountWidth = narrow ? 54.0 : 66.0;
        return Container(
          key: ValueKey<String>('http-connection-lane-$index'),
          height: 38,
          decoration: BoxDecoration(
            border: Border(bottom: BorderSide(color: palette.border)),
          ),
          child: Row(
            children: [
              SizedBox(
                width: 28,
                child: Text(
                  '#${index + 1}',
                  style: TextStyle(color: palette.textPrimary, fontSize: 11, fontWeight: FontWeight.w600),
                ),
              ),
              SizedBox(
                key: ValueKey<String>('http-connection-status-$index'),
                width: 18,
                child: Center(
                  child: _ConnectionStatusIcon(
                    status: status,
                    color: statusColor,
                    retryTimes: connection.retryTimes,
                    failed: connection.failed,
                    retrying: retrying,
                    completed: connection.completed,
                  ),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: TaskProgressBar(
                  key: ValueKey<String>('http-connection-progress-$index'),
                  value: progress,
                  indeterminate: downloading && !hasTotal,
                  shimmer: downloading,
                  height: 7,
                  trackColor: palette.brandTrack,
                  fillColor: palette.brandProgress,
                  highlightStartColor: palette.brandProgress,
                  highlightEndColor: palette.brandProgress,
                ),
              ),
              const SizedBox(width: 8),
              SizedBox(
                width: amountWidth,
                child: Text(
                  _formatBytes(connection.downloaded),
                  textAlign: TextAlign.right,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(color: palette.textSecondary, fontSize: narrow ? 10 : 11),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

class _ConnectionStatusIcon extends StatelessWidget {
  const _ConnectionStatusIcon({
    required this.status,
    required this.color,
    required this.retryTimes,
    required this.failed,
    required this.retrying,
    required this.completed,
  });

  final String status;
  final Color color;
  final int retryTimes;
  final bool failed;
  final bool retrying;
  final bool completed;

  @override
  Widget build(BuildContext context) {
    final icon = Semantics(
      label: status,
      child: Icon(
        failed
            ? Icons.error_rounded
            : retrying
            ? Icons.sync_problem_rounded
            : completed
            ? Icons.check_circle_rounded
            : Icons.downloading_rounded,
        size: 15,
        color: color,
      ),
    );
    if (!failed && !retrying) return icon;
    final message = failed
        ? retryTimes > 0
              ? context.l10n.connectionFailedRetries(retryTimes)
              : context.l10n.connectionFailed
        : context.l10n.retryingCount(retryTimes);
    return AppTooltip(message: message, child: icon);
  }
}

class _BtStatistics extends StatelessWidget {
  const _BtStatistics({required this.stats, required this.task});

  final BtTaskStats stats;
  final TaskRecord task;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionTitle(context.l10n.pieceStatus),
        const SizedBox(height: 16),
        PieceMap(pieceMap: stats.pieceMap, totalBytes: task.totalBytes),
        const _PieceLegend(),
        const _SectionDivider(),
        _SummaryGrid(
          items: [
            _SummaryItem(context.l10n.peers, '${stats.activePeers} / ${stats.totalPeers}'),
            _SummaryItem(context.l10n.seeders, stats.connectedSeeders.toString()),
            _SummaryItem(context.l10n.leechers, stats.connectedLeechers.toString()),
            _SummaryItem(context.l10n.uploadedAmount, _formatBytes(stats.seedBytes)),
            _SummaryItem(context.l10n.shareRatio, stats.seedRatio.toStringAsFixed(2)),
            _SummaryItem(
              context.l10n.shareDuration,
              formatTaskDuration(context.l10n, Duration(seconds: stats.seedTime)),
            ),
          ],
        ),
        const _SectionDivider(),
        _SectionTitle('${context.l10n.peers}  ${context.l10n.connectedPeers(stats.activePeers)}'),
        const SizedBox(height: 12),
        PeerTable(protocol: PeerTableProtocol.bt, peers: stats.peers),
      ],
    );
  }
}

class _Ed2kStatistics extends StatelessWidget {
  const _Ed2kStatistics({required this.stats});

  final Ed2kTaskStats stats;

  @override
  Widget build(BuildContext context) {
    final completedPieces = stats.pieceMap.isEmpty ? '—' : stats.completedPieces.toString();
    final completed = stats.totalWanted > 0
        ? '${((stats.totalDone / stats.totalWanted).clamp(0.0, 1.0) * 100).round()}%'
        : '—';
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionTitle(context.l10n.pieceStatus),
        const SizedBox(height: 16),
        PieceMap(pieceMap: stats.pieceMap, totalBytes: stats.totalWanted > 0 ? stats.totalWanted : null),
        const _PieceLegend(),
        const _SectionDivider(),
        _SummaryGrid(
          items: [
            _SummaryItem(context.l10n.peers, stats.totalPeers.toString()),
            _SummaryItem(context.l10n.active, stats.activePeers.toString()),
            _SummaryItem(context.l10n.completedPieces, completedPieces),
            _SummaryItem(context.l10n.speed, _formatRate(stats.downloadRate)),
            _SummaryItem(context.l10n.uploadSpeed, _formatRate(stats.uploadRate)),
            _SummaryItem(context.l10n.completed, completed),
          ],
        ),
        const _SectionDivider(),
        _SectionTitle('${context.l10n.peers}  ${context.l10n.activePeers(stats.activePeers)}'),
        const SizedBox(height: 12),
        PeerTable(protocol: PeerTableProtocol.ed2k, peers: stats.peers),
      ],
    );
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle(this.label, {this.trailing});

  final String label;
  final String? trailing;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      children: [
        Expanded(
          child: Text(
            label,
            style: TextStyle(color: palette.textPrimary, fontSize: 14, fontWeight: FontWeight.w700),
          ),
        ),
        if (trailing != null)
          Text(
            trailing!,
            style: TextStyle(color: palette.textMuted, fontSize: 11, fontWeight: FontWeight.w500),
          ),
      ],
    );
  }
}

class _SectionDivider extends StatelessWidget {
  const _SectionDivider();

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(height: 1, margin: const EdgeInsets.symmetric(vertical: 22), color: palette.border);
  }
}

class _SummaryItem {
  const _SummaryItem(this.label, this.value);

  final String label;
  final String value;
}

class _SummaryGrid extends StatelessWidget {
  const _SummaryGrid({required this.items});

  final List<_SummaryItem> items;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return LayoutBuilder(
      builder: (context, constraints) {
        final columns = constraints.maxWidth >= 280 ? 3 : 2;
        final width = constraints.maxWidth / columns;
        return Wrap(
          runSpacing: 18,
          children: [
            for (var index = 0; index < items.length; index++)
              SizedBox(
                width: width,
                child: Column(
                  crossAxisAlignment: switch (index % columns) {
                    0 => CrossAxisAlignment.start,
                    final column when column == columns - 1 => CrossAxisAlignment.end,
                    _ => CrossAxisAlignment.center,
                  },
                  children: [
                    Text(items[index].label, style: TextStyle(color: palette.textMuted, fontSize: 11)),
                    const SizedBox(height: 5),
                    Text(
                      items[index].value,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(color: palette.textPrimary, fontSize: 15, fontWeight: FontWeight.w700),
                    ),
                  ],
                ),
              ),
          ],
        );
      },
    );
  }
}

class _PieceLegend extends StatelessWidget {
  const _PieceLegend();

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Wrap(
      spacing: 18,
      runSpacing: 8,
      children: [
        _LegendItem(label: context.l10n.incomplete, color: palette.surfaceSoft),
        _LegendItem(label: context.l10n.completed, color: palette.success),
      ],
    );
  }
}

class _LegendItem extends StatelessWidget {
  const _LegendItem({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(
          width: 9,
          height: 9,
          child: ClipRRect(
            borderRadius: BorderRadius.circular(2),
            child: ColoredBox(color: color),
          ),
        ),
        const SizedBox(width: 6),
        Text(label, style: TextStyle(color: palette.textSecondary, fontSize: 11)),
      ],
    );
  }
}

class _StatisticsSkeleton extends StatefulWidget {
  const _StatisticsSkeleton();

  @override
  State<_StatisticsSkeleton> createState() => _StatisticsSkeletonState();
}

class _StatisticsSkeletonState extends State<_StatisticsSkeleton> with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(vsync: this, duration: const Duration(milliseconds: 1200))..repeat(reverse: true);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, _) {
        final color = palette.surfaceSoft.withValues(alpha: 0.55 + _controller.value * 0.35);
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _SkeletonLine(width: 88, height: 16, color: color),
            const SizedBox(height: 22),
            _SkeletonLine(width: double.infinity, height: 74, color: color),
            const SizedBox(height: 20),
            _SkeletonLine(width: double.infinity, height: 110, color: color),
          ],
        );
      },
    );
  }
}

class _SkeletonLine extends StatelessWidget {
  const _SkeletonLine({required this.width, required this.height, required this.color});

  final double width;
  final double height;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: width,
      height: height,
      decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(4)),
    );
  }
}

class _StatisticsEmptyState extends StatelessWidget {
  const _StatisticsEmptyState();

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Center(
      child: Text(context.l10n.noStatistics, style: TextStyle(color: palette.textMuted, fontSize: 13)),
    );
  }
}

String _formatRate(int bytesPerSecond) => bytesPerSecond <= 0 ? '—' : '${_formatBytes(bytesPerSecond)}/s';

String _formatBytes(int bytes) {
  return ByteSizeFormatter.format(bytes);
}
