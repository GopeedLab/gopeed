import 'dart:math';

import 'package:flutter/material.dart' show Icons;
import 'package:flutter/physics.dart';
import 'package:flutter/widgets.dart';

import '../../../../core/utils/transfer_rate_formatter.dart';
import '../../../../shared/theme/app_palette.dart';

class SpeedMonitorCard extends StatefulWidget {
  const SpeedMonitorCard({super.key, this.downloadBytesPerSecond = 0, this.uploadBytesPerSecond = 0});

  final int downloadBytesPerSecond;
  final int uploadBytesPerSecond;

  @override
  State<SpeedMonitorCard> createState() => _SpeedMonitorCardState();
}

class _SpeedMonitorCardState extends State<SpeedMonitorCard> with SingleTickerProviderStateMixin {
  static const _needleSpring = SpringDescription(mass: 1, stiffness: 140, damping: 11);
  static const int _sampleWindowSize = 30;
  static const double _maxNeedleLaunchVelocity = 4.4;
  static const double _minVelocityFloor = 0.18;
  static const double _displayHeadroom = 1.2;
  static const double _minDisplayMax = 12;
  static const double _maxDisplayMax = 100;
  static const double _displayDecayFactor = 0.965;
  static const double _highPressureThreshold = 0.9;
  static const int _highPressureWindow = 2;
  static const double _highPressureBoost = 1.12;

  final List<double> _downloadSamples = <double>[];
  late final AnimationController _needleController;
  late double _download;
  late double _displayMax;
  int _highPressureStreak = 0;

  @override
  void initState() {
    super.initState();
    _download = _toMegabytesPerSecond(widget.downloadBytesPerSecond);
    _displayMax = max(_minDisplayMax, _download * _displayHeadroom);
    _primeSamples(_download);
    _needleController = AnimationController.unbounded(vsync: this, value: (_download / _displayMax).clamp(0.0, 1.0))
      ..addListener(() {
        if (mounted) {
          setState(() {});
        }
      });
  }

  @override
  void didUpdateWidget(SpeedMonitorCard oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.downloadBytesPerSecond == widget.downloadBytesPerSecond) {
      return;
    }
    _setDownloadSpeed(_toMegabytesPerSecond(widget.downloadBytesPerSecond));
  }

  void _setDownloadSpeed(double download) {
    final previousDownload = _download;
    final nextDisplayMax = _resolveDisplayMax(download);
    _download = download;
    _displayMax = nextDisplayMax;
    final target = (download / nextDisplayMax).clamp(0.0, 1.0);
    final launchVelocity = _computeNeedleLaunchVelocity(previousDownload: previousDownload, nextDownload: download);
    _needleController.animateWith(SpringSimulation(_needleSpring, _needleController.value, target, launchVelocity));
  }

  void _primeSamples(double initialValue) {
    _downloadSamples
      ..clear()
      ..add(initialValue);
    _highPressureStreak = 0;
  }

  double _resolveDisplayMax(double currentDownload) {
    _downloadSamples.add(currentDownload);
    if (_downloadSamples.length > _sampleWindowSize) {
      _downloadSamples.removeAt(0);
    }

    final currentUsage = _displayMax <= 0 ? 0.0 : (currentDownload / _displayMax).clamp(0.0, 2.0);
    if (currentUsage >= _highPressureThreshold) {
      _highPressureStreak += 1;
    } else {
      _highPressureStreak = 0;
    }

    final sorted = List<double>.from(_downloadSamples)..sort();
    final p95Index = ((sorted.length - 1) * 0.95).round().clamp(0, sorted.length - 1);
    final percentile95 = sorted[p95Index];
    var targetMax = (percentile95 * _displayHeadroom).clamp(_minDisplayMax, _maxDisplayMax);

    if (_highPressureStreak >= _highPressureWindow) {
      targetMax = max(targetMax, (_displayMax * _highPressureBoost).clamp(_minDisplayMax, _maxDisplayMax));
    }

    if (targetMax >= _displayMax) {
      return targetMax;
    }

    // Expand quickly, contract slowly, so the gauge does not jitter when
    // bandwidth fluctuates downward for a moment.
    return max(targetMax, _displayMax * _displayDecayFactor);
  }

  double _computeNeedleLaunchVelocity({required double previousDownload, required double nextDownload}) {
    final delta = nextDownload - previousDownload;
    if (delta.abs() < 0.01) {
      return 0;
    }

    if (delta < 0) {
      // Falling speeds should settle quickly instead of overshooting downward.
      return min(_needleController.velocity, 0) * 0.22;
    }

    final combined = previousDownload + nextDownload;
    final relativeJump = combined <= 0 ? 1.0 : delta / combined;
    final absoluteJump = delta / 100.0;

    // Relative jump adapts to slow and fast networks; absolute jump prevents
    // large moves at high speeds from feeling underpowered.
    final jumpFactor = (relativeJump * 0.72) + (sqrt(absoluteJump.clamp(0.0, 1.0)) * 0.28);
    final normalized = jumpFactor.clamp(0.0, 1.0);
    final launchVelocity = _minVelocityFloor + (_maxNeedleLaunchVelocity - _minVelocityFloor) * normalized;

    // Keep a little of the current momentum so rapid consecutive spikes feel continuous.
    final carriedMomentum = max(_needleController.velocity, 0) * 0.2;
    return min(_maxNeedleLaunchVelocity, max(launchVelocity, carriedMomentum));
  }

  @override
  void dispose() {
    _needleController.dispose();
    super.dispose();
  }

  double _toMegabytesPerSecond(int bytesPerSecond) => max(0, bytesPerSecond) / (1024 * 1024);

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final gaugeForeground = palette.textPrimary;
    final arcColor = gaugeForeground;
    final needleColor = gaugeForeground;
    final trackColor = gaugeForeground.withValues(alpha: 0.12);
    final tickColor = gaugeForeground.withValues(alpha: 0.18);
    Widget buildGauge() {
      return SizedBox(
        key: const ValueKey('speed-monitor-gauge'),
        width: 60,
        height: 32,
        child: TweenAnimationBuilder<double>(
          tween: Tween<double>(begin: 0, end: (_download / _displayMax).clamp(0.0, 1.0)),
          duration: const Duration(milliseconds: 650),
          curve: Curves.easeOutCubic,
          builder: (context, value, _) {
            return CustomPaint(
              painter: _SpeedGaugePainter(
                value: value.clamp(0.0, 1.0),
                needleValue: _needleController.value.clamp(-0.08, 1.08),
                arcColor: arcColor,
                needleColor: needleColor,
                trackColor: trackColor,
                tickColor: tickColor,
              ),
            );
          },
        ),
      );
    }

    Widget buildSpeedLines({required bool compact}) {
      return Column(
        key: const ValueKey('speed-monitor-values'),
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          _SpeedLine(
            key: const ValueKey('speed-monitor-download-line'),
            icon: Icons.south,
            bytesPerSecond: widget.downloadBytesPerSecond,
            foreground: palette.textPrimary,
            unitColor: palette.textSecondary,
            compact: compact,
            valueSize: compact ? 12 : 13,
          ),
          SizedBox(height: compact ? 3 : 4),
          _SpeedLine(
            key: const ValueKey('speed-monitor-upload-line'),
            icon: Icons.north,
            bytesPerSecond: widget.uploadBytesPerSecond,
            foreground: palette.textSecondary,
            unitColor: palette.textMuted,
            compact: compact,
            valueSize: compact ? 10 : 11,
          ),
        ],
      );
    }

    return AnimatedContainer(
      duration: const Duration(milliseconds: 180),
      margin: const EdgeInsets.fromLTRB(8, 0, 8, 24),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 12),
      decoration: BoxDecoration(
        color: palette.cardBg.withValues(alpha: 0.72),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: palette.border),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          if (constraints.maxWidth < 170) {
            final gauge = buildGauge();
            return Row(
              children: [
                gauge,
                const SizedBox(width: 6),
                Expanded(child: buildSpeedLines(compact: true)),
              ],
            );
          }
          final gauge = buildGauge();
          return Row(
            children: [
              gauge,
              const SizedBox(width: 14),
              Expanded(child: buildSpeedLines(compact: false)),
            ],
          );
        },
      ),
    );
  }
}

class _SpeedLine extends StatelessWidget {
  const _SpeedLine({
    super.key,
    required this.icon,
    required this.bytesPerSecond,
    required this.foreground,
    required this.unitColor,
    this.valueSize = 13,
    this.compact = false,
  });

  final IconData icon;
  final int bytesPerSecond;
  final Color foreground;
  final Color unitColor;
  final double valueSize;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final display = TransferRateFormatter.format(bytesPerSecond);
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(icon, size: compact ? 8 : 10, color: foreground),
        SizedBox(width: compact ? 2 : 4),
        Text(
          display.value,
          maxLines: 1,
          style: TextStyle(color: foreground, fontSize: valueSize, fontWeight: FontWeight.w700, height: 1),
        ),
        SizedBox(width: compact ? 2 : 4),
        Text(
          display.unit,
          maxLines: 1,
          style: TextStyle(
            color: unitColor,
            fontSize: compact ? 8 : 9,
            fontWeight: FontWeight.w600,
            letterSpacing: compact ? 0 : 1.2,
            height: 1,
          ),
        ),
      ],
    );
  }
}

class _SpeedGaugePainter extends CustomPainter {
  const _SpeedGaugePainter({
    required this.value,
    required this.needleValue,
    required this.arcColor,
    required this.needleColor,
    required this.trackColor,
    required this.tickColor,
  });

  final double value;
  final double needleValue;
  final Color arcColor;
  final Color needleColor;
  final Color trackColor;
  final Color tickColor;

  @override
  void paint(Canvas canvas, Size size) {
    final radius = min(size.width / 2 - 6, size.height - 6);
    final center = Offset(size.width / 2, (size.height + radius) / 2);
    final rect = Rect.fromCircle(center: center, radius: radius);
    final tickPaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round
      ..strokeWidth = 1.4
      ..color = tickColor;
    final trackPaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3
      ..strokeCap = StrokeCap.round
      ..color = trackColor;
    final activePaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3
      ..strokeCap = StrokeCap.round
      ..color = arcColor;

    const tickCount = 5;
    for (var i = 0; i < tickCount; i++) {
      final t = i / (tickCount - 1);
      final tickAngle = pi + (pi * t);
      final outerRadius = radius - 2;
      final innerRadius = outerRadius - 4;
      final start = Offset(center.dx + cos(tickAngle) * innerRadius, center.dy + sin(tickAngle) * innerRadius);
      final end = Offset(center.dx + cos(tickAngle) * outerRadius, center.dy + sin(tickAngle) * outerRadius);
      canvas.drawLine(start, end, tickPaint);
    }

    canvas.drawArc(rect, pi, pi, false, trackPaint);
    canvas.drawArc(rect, pi, pi * value, false, activePaint);

    final angle = pi + (pi * needleValue);
    final needleEnd = Offset(center.dx + cos(angle) * (radius - 6), center.dy + sin(angle) * (radius - 6));
    final needlePaint = Paint()
      ..color = needleColor
      ..strokeWidth = 1.6
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(center, needleEnd, needlePaint);
    canvas.drawCircle(center, 2.6, Paint()..color = needleColor);
  }

  @override
  bool shouldRepaint(covariant _SpeedGaugePainter oldDelegate) {
    return oldDelegate.value != value ||
        oldDelegate.needleValue != needleValue ||
        oldDelegate.arcColor != arcColor ||
        oldDelegate.needleColor != needleColor ||
        oldDelegate.trackColor != trackColor ||
        oldDelegate.tickColor != tickColor;
  }
}
