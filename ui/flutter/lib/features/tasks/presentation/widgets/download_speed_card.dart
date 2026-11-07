import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/utils/transfer_rate_formatter.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../application/download_speed_history_provider.dart';
import '../../domain/download_speed_history.dart';
import '../../domain/task_record.dart';
import '../task_record_localizations.dart';

class DownloadSpeedCard extends ConsumerWidget {
  const DownloadSpeedCard({super.key, required this.task});

  final TaskRecord task;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final palette = AppPalette.of(context);
    final history = ref.watch(downloadSpeedHistoryProvider(task.id)).value ?? const DownloadSpeedHistory();
    final average = task.averageSpeedBytes;
    final speed = task.status == TaskStatus.downloading && !task.waiting ? task.speed : '0 B/s';
    final ceiling = history.peak > 0 ? history.peak : 1;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: AppDesignTokens.space12, vertical: AppDesignTokens.space8),
      decoration: BoxDecoration(
        color: palette.cardBg,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(AppDesignTokens.windowRadius),
      ),
      child: Column(
        children: [
          Row(
            children: [
              Expanded(child: Text(context.l10n.downloadSpeedHistory).small().semiBold()),
              const SizedBox(width: AppDesignTokens.space8),
              Expanded(
                child: Text(
                  formatTaskDuration(context.l10n, history.elapsed),
                  textAlign: TextAlign.end,
                ).xSmall().muted(),
              ),
            ],
          ),
          const SizedBox(height: AppDesignTokens.space4),
          const Divider(),
          const SizedBox(height: AppDesignTokens.space4),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: _Metric(label: context.l10n.speed, value: speed ?? '—', color: palette.textPrimary),
              ),
              Expanded(
                child: _Metric(label: context.l10n.remaining, value: task.localizedRemaining(context.l10n) ?? '—'),
              ),
              Expanded(
                child: _Metric(
                  label: context.l10n.averageSpeed,
                  value: average == null ? '—' : TransferRateFormatter.format(average).text,
                ),
              ),
            ],
          ),
          const SizedBox(height: AppDesignTokens.space4),
          SizedBox(
            height: AppDesignTokens.downloadSpeedChartHeight,
            child: Row(
              children: [
                Expanded(
                  child: Semantics(
                    label: context.l10n.downloadSpeedHistory,
                    value: '${formatTaskDuration(context.l10n, history.elapsed)}: ${speed ?? '—'}',
                    child: CustomPaint(
                      key: const ValueKey('download-speed-chart'),
                      size: const Size(double.infinity, AppDesignTokens.downloadSpeedChartHeight),
                      painter: DownloadSpeedChartPainter(
                        history: history,
                        ceiling: ceiling,
                        color: palette.taskCardProgressFill,
                        gridColor: palette.border,
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: AppDesignTokens.space8),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(history.peak == 0 ? '—' : TransferRateFormatter.format(ceiling).text).xSmall().muted(),
                    Text(history.peak == 0 ? '—' : TransferRateFormatter.format(ceiling ~/ 2).text).xSmall().muted(),
                    const Text('0').xSmall().muted(),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _Metric extends StatelessWidget {
  const _Metric({required this.label, required this.value, this.color});
  final String label;
  final String value;
  final Color? color;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsetsDirectional.only(end: AppDesignTokens.space8),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label).xSmall().muted(),
        const SizedBox(height: AppDesignTokens.space4),
        Text(
          value,
          style: TextStyle(color: color),
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
        ).small().semiBold(),
      ],
    ),
  );
}

class DownloadSpeedChartPainter extends CustomPainter {
  const DownloadSpeedChartPainter({
    required this.history,
    required this.ceiling,
    required this.color,
    required this.gridColor,
  });
  final DownloadSpeedHistory history;
  final int ceiling;
  final Color color;
  final Color gridColor;

  @override
  void paint(Canvas canvas, Size size) {
    final grid = Paint()
      ..color = gridColor
      ..strokeWidth = 1;
    for (final fraction in [0.0, 0.5, 1.0]) {
      final y = 2 + (size.height - 4) * fraction;
      for (double x = 0; x < size.width; x += 9) {
        canvas.drawLine(Offset(x, y), Offset((x + 4).clamp(0, size.width), y), grid);
      }
    }
    final samples = history.samples;
    if (samples.isEmpty) return;
    final line = Paint()
      ..color = color
      ..strokeWidth = 2
      ..style = PaintingStyle.stroke;
    final fill = Paint()
      ..shader = LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [color.withValues(alpha: .28), color.withValues(alpha: 0)],
      ).createShader(Offset.zero & size);
    Path path = Path();
    Offset? first;
    Offset? last;
    void finishSegment() {
      if (first == null || last == null) return;
      final area = Path.from(path)
        ..lineTo(last.dx, size.height)
        ..lineTo(first.dx, size.height)
        ..close();
      canvas.drawPath(area, fill);
      canvas.drawPath(path, line);
      canvas.drawCircle(last, 2, Paint()..color = color);
    }

    for (var i = 0; i < samples.length; i++) {
      final sample = samples[i];
      final x = size.width * history.position(sample.time);
      final point = Offset(x.clamp(0, size.width), 2 + (size.height - 4) * (1 - sample.bytesPerSecond / ceiling));
      // API outages leave gaps instead of implying a measured speed.
      if (i == 0 || sample.time.difference(samples[i - 1].time) > const Duration(seconds: 3)) {
        finishSegment();
        path = Path()..moveTo(point.dx, point.dy);
        first = point;
      } else {
        path.lineTo(point.dx, point.dy);
      }
      last = point;
    }
    finishSegment();
  }

  @override
  bool shouldRepaint(DownloadSpeedChartPainter oldDelegate) =>
      oldDelegate.history != history ||
      oldDelegate.color != color ||
      oldDelegate.gridColor != gridColor ||
      oldDelegate.ceiling != ceiling;
}
