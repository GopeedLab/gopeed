import 'dart:async';
import 'dart:math' as math;
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

/// Simulated download speed sample data (35 points, ~100-5000 KB/s range) for visual prototype demonstration.
const List<double> mockSpeedSamples = [
  150000.0, 320000.0, 680000.0, 1200000.0, 2100000.0, 2800000.0, 3400000.0,
  3100000.0, 2900000.0, 3300000.0, 3800000.0, 4200000.0, 4100000.0, 4500000.0,
  4900000.0, 4700000.0, 4800000.0, 4600000.0, 4300000.0, 4500000.0, 4800000.0,
  5000000.0, 4700000.0, 4300000.0, 3900000.0, 4100000.0, 4400000.0, 4700000.0,
  4900000.0, 4800000.0, 4500000.0, 4200000.0, 3700000.0, 3100000.0, 2800000.0,
];

/// A robust, standalone live download speed line chart with edge-case hardening.
///
/// Renders a rolling speed-over-time line chart using Flutter's native Canvas / CustomPainter.
class TaskSpeedChart extends StatefulWidget {
  /// The list of speed samples in bytes/second. Defaults to [mockSpeedSamples] for demonstration.
  final List<double> speedSamples;

  /// Optional fixed height for the chart container.
  final double? height;

  /// Optional fixed width for the chart container.
  final double? width;

  /// Line color. Defaults to [ColorScheme.primary].
  final Color? lineColor;

  /// Fill area color under the line. Defaults to [ColorScheme.primary] with low opacity.
  final Color? fillColor;

  /// Line stroke width. Defaults to 2.0.
  final double strokeWidth;

  /// Whether to render a gradient fill area under the line. Defaults to true.
  final bool showFill;

  /// Whether the task is currently paused. When true, visually dims the chart.
  final bool isPaused;

  /// Whether the task has reached completed (Done) status.
  final bool isCompleted;

  /// Minimum duration between visual chart re-renders to prevent excessive repainting.
  final Duration throttleDuration;

  /// Optional custom widget to display when there are fewer than 2 data points.
  final Widget? emptyPlaceholder;

  const TaskSpeedChart({
    Key? key,
    this.speedSamples = const <double>[],
    this.height,
    this.width,
    this.lineColor,
    this.fillColor,
    this.strokeWidth = 2.0,
    this.showFill = true,
    this.isPaused = false,
    this.isCompleted = false,
    this.throttleDuration = const Duration(milliseconds: 500),
    this.emptyPlaceholder,
  }) : super(key: key);

  @override
  State<TaskSpeedChart> createState() => _TaskSpeedChartState();
}

class _TaskSpeedChartState extends State<TaskSpeedChart> {
  late List<double> _renderedSamples;
  Timer? _throttleTimer;
  DateTime? _lastRenderTime;

  @override
  void initState() {
    super.initState();
    _renderedSamples = List<double>.from(widget.speedSamples);
    _lastRenderTime = DateTime.now();
  }

  @override
  void didUpdateWidget(covariant TaskSpeedChart oldWidget) {
    super.didUpdateWidget(oldWidget);

    final samplesChanged = !listEquals(widget.speedSamples, _renderedSamples);
    final stateChanged = widget.isPaused != oldWidget.isPaused ||
        widget.isCompleted != oldWidget.isCompleted ||
        widget.lineColor != oldWidget.lineColor ||
        widget.fillColor != oldWidget.fillColor;

    if (!samplesChanged && !stateChanged) {
      return;
    }

    if (stateChanged || widget.throttleDuration == Duration.zero) {
      _throttleTimer?.cancel();
      _renderedSamples = List<double>.from(widget.speedSamples);
      _lastRenderTime = DateTime.now();
      return;
    }

    final now = DateTime.now();
    final elapsed = _lastRenderTime == null
        ? widget.throttleDuration
        : now.difference(_lastRenderTime!);

    if (elapsed >= widget.throttleDuration) {
      _throttleTimer?.cancel();
      _renderedSamples = List<double>.from(widget.speedSamples);
      _lastRenderTime = now;
    } else {
      if (_throttleTimer == null || !_throttleTimer!.isActive) {
        final remaining = widget.throttleDuration - elapsed;
        _throttleTimer = Timer(remaining, () {
          if (mounted) {
            setState(() {
              _renderedSamples = List<double>.from(widget.speedSamples);
              _lastRenderTime = DateTime.now();
            });
          }
        });
      }
    }
  }

  @override
  void dispose() {
    _throttleTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    // Determine colors based on task state
    Color baseLineColor;
    if (widget.lineColor != null) {
      baseLineColor = widget.lineColor!;
    } else if (widget.isCompleted) {
      baseLineColor = Colors.green;
    } else {
      baseLineColor = theme.colorScheme.primary;
    }

    // Dim visual appearance if paused
    final effectiveLineColor = widget.isPaused
        ? baseLineColor.withOpacity(0.4)
        : baseLineColor;

    final baseFillColor = widget.fillColor ?? baseLineColor.withOpacity(0.15);
    final effectiveFillColor = widget.isPaused
        ? baseFillColor.withOpacity(0.05)
        : baseFillColor;

    Widget content;

    if (_renderedSamples.length < 2) {
      if (widget.emptyPlaceholder != null) {
        content = widget.emptyPlaceholder!;
      } else {
        content = CustomPaint(
          size: Size.infinite,
          painter: _EmptySpeedChartPainter(
            singleSample:
                _renderedSamples.isNotEmpty ? _renderedSamples.first : null,
            color: effectiveLineColor.withOpacity(0.3),
            strokeWidth: widget.strokeWidth,
          ),
        );
      }
    } else {
      content = CustomPaint(
        size: Size.infinite,
        painter: _SpeedChartPainter(
          samples: _renderedSamples,
          lineColor: effectiveLineColor,
          fillColor: effectiveFillColor,
          strokeWidth: widget.strokeWidth,
          showFill: widget.showFill,
        ),
      );
    }

    return SizedBox(
      height: widget.height,
      width: widget.width,
      child: content,
    );
  }
}

class _SpeedChartPainter extends CustomPainter {
  final List<double> samples;
  final Color lineColor;
  final Color fillColor;
  final double strokeWidth;
  final bool showFill;

  _SpeedChartPainter({
    required this.samples,
    required this.lineColor,
    required this.fillColor,
    required this.strokeWidth,
    required this.showFill,
  });

  @override
  void paint(Canvas canvas, Size size) {
    if (size.width <= 0 || size.height <= 0 || samples.length < 2) {
      return;
    }

    // Sanitize samples to prevent NaN or Infinity values
    final cleanSamples = samples.map((s) {
      if (s.isNaN || s.isInfinite || s < 0) return 0.0;
      return s;
    }).toList();

    double maxVal = cleanSamples.fold(0.0, (prev, val) => math.max(prev, val));
    if (maxVal <= 0) {
      maxVal = 1.0;
    }

    final double topPadding = strokeWidth + 2.0;
    final double bottomPadding = strokeWidth;
    final double chartHeight = size.height - topPadding - bottomPadding;

    final points = <Offset>[];
    final int count = cleanSamples.length;
    final double dx = size.width / (count - 1);

    for (int i = 0; i < count; i++) {
      final double x = i * dx;
      final double normalized = cleanSamples[i] / maxVal;
      final double y = topPadding + (1.0 - normalized) * chartHeight;
      points.add(Offset(x, y));
    }

    // Build the line path
    final linePath = Path();
    linePath.moveTo(points[0].dx, points[0].dy);

    for (int i = 1; i < points.length; i++) {
      linePath.lineTo(points[i].dx, points[i].dy);
    }

    // Draw gradient fill under the line if enabled
    if (showFill) {
      final fillPath = Path.from(linePath);
      fillPath.lineTo(points.last.dx, size.height);
      fillPath.lineTo(points.first.dx, size.height);
      fillPath.close();

      final fillPaint = Paint()
        ..style = PaintingStyle.fill
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [
            fillColor,
            fillColor.withOpacity(0.0),
          ],
        ).createShader(Rect.fromLTWH(0, 0, size.width, size.height));

      canvas.drawPath(fillPath, fillPaint);
    }

    // Draw the speed curve line
    final linePaint = Paint()
      ..color = lineColor
      ..style = PaintingStyle.stroke
      ..strokeWidth = strokeWidth
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round
      ..isAntiAlias = true;

    canvas.drawPath(linePath, linePaint);
  }

  @override
  bool shouldRepaint(covariant _SpeedChartPainter oldDelegate) {
    if (oldDelegate.samples.length != samples.length ||
        oldDelegate.lineColor != lineColor ||
        oldDelegate.fillColor != fillColor ||
        oldDelegate.strokeWidth != strokeWidth ||
        oldDelegate.showFill != showFill) {
      return true;
    }
    for (int i = 0; i < samples.length; i++) {
      if (oldDelegate.samples[i] != samples[i]) {
        return true;
      }
    }
    return false;
  }
}

class _EmptySpeedChartPainter extends CustomPainter {
  final double? singleSample;
  final Color color;
  final double strokeWidth;

  _EmptySpeedChartPainter({
    this.singleSample,
    required this.color,
    required this.strokeWidth,
  });

  @override
  void paint(Canvas canvas, Size size) {
    if (size.width <= 0 || size.height <= 0) return;

    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = strokeWidth
      ..strokeCap = StrokeCap.round;

    final double y = size.height - strokeWidth;

    if (singleSample != null && singleSample! > 0) {
      // Draw a subtle baseline across the width for single sample
      canvas.drawLine(Offset(0, y), Offset(size.width, y), paint);
    } else {
      // Subtle baseline for empty state
      canvas.drawLine(Offset(0, y), Offset(size.width, y), paint);
    }
  }

  @override
  bool shouldRepaint(covariant _EmptySpeedChartPainter oldDelegate) {
    return oldDelegate.singleSample != singleSample ||
        oldDelegate.color != color ||
        oldDelegate.strokeWidth != strokeWidth;
  }
}
