import 'package:flutter/widgets.dart';

class TaskProgressBar extends StatefulWidget {
  const TaskProgressBar({
    super.key,
    required this.trackColor,
    required this.fillColor,
    required this.highlightStartColor,
    required this.highlightEndColor,
    this.value,
    this.indeterminate = false,
    this.shimmer = true,
    this.height = 2,
  });

  final Color trackColor;
  final Color fillColor;
  final Color highlightStartColor;
  final Color highlightEndColor;
  final double? value;
  final bool indeterminate;
  final bool shimmer;
  final double height;

  @override
  State<TaskProgressBar> createState() => _TaskProgressBarState();
}

class _TaskProgressBarState extends State<TaskProgressBar> with SingleTickerProviderStateMixin {
  static const double _shimmerPixelsPerSecond = 140.0;
  static const double _shimmerMinDurationMs = 900;
  static const double _shimmerMaxDurationMs = 3200;

  late final AnimationController _controller;

  bool get _shouldAnimate => widget.indeterminate || widget.shimmer;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(vsync: this, duration: const Duration(milliseconds: 1600));
    _syncAnimation();
  }

  @override
  void didUpdateWidget(covariant TaskProgressBar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.indeterminate != oldWidget.indeterminate || widget.shimmer != oldWidget.shimmer) {
      _syncAnimation();
    }
  }

  void _syncAnimation() {
    if (_shouldAnimate) {
      if (!_controller.isAnimating) _controller.repeat();
    } else {
      _controller.stop();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: widget.height,
      child: ClipRRect(
        borderRadius: BorderRadius.circular(999),
        child: ColoredBox(
          color: widget.trackColor,
          child: LayoutBuilder(
            builder: (context, constraints) {
              return AnimatedBuilder(
                animation: _controller,
                builder: (context, child) {
                  if (widget.indeterminate) {
                    final segmentWidth = constraints.maxWidth * 0.4;
                    final left = lerpDouble(-segmentWidth, constraints.maxWidth, _controller.value);

                    return Stack(
                      fit: StackFit.expand,
                      children: [
                        Positioned(
                          left: left,
                          width: segmentWidth,
                          top: 0,
                          bottom: 0,
                          child: _ProgressFill(
                            color: widget.fillColor,
                            shimmer: widget.shimmer,
                            highlightStartColor: widget.highlightStartColor,
                            highlightEndColor: widget.highlightEndColor,
                            bandLeft: lerpDouble(-segmentWidth, segmentWidth * 2, _controller.value),
                            bandWidth: segmentWidth * 0.5,
                          ),
                        ),
                      ],
                    );
                  }

                  final fillWidth = constraints.maxWidth * ((widget.value ?? 0).clamp(0, 1));
                  final bandWidth = fillWidth == 0 ? 0.0 : (fillWidth * 0.35).clamp(16.0, 40.0);
                  final travelDistance = fillWidth + bandWidth * 2;
                  final durationMs = (travelDistance / _shimmerPixelsPerSecond * 1000)
                      .clamp(_shimmerMinDurationMs, _shimmerMaxDurationMs)
                      .round();

                  if (widget.shimmer && widget.value != null && fillWidth > 0) {
                    final duration = Duration(milliseconds: durationMs);
                    if (_controller.duration != duration) {
                      _controller.duration = duration;
                    }
                  }

                  return Stack(
                    fit: StackFit.expand,
                    children: [
                      Align(
                        alignment: Alignment.centerLeft,
                        child: SizedBox(
                          width: fillWidth,
                          child: _ProgressFill(
                            color: widget.fillColor,
                            shimmer: widget.shimmer,
                            highlightStartColor: widget.highlightStartColor,
                            highlightEndColor: widget.highlightEndColor,
                            bandLeft: lerpDouble(-bandWidth, fillWidth + bandWidth, _controller.value),
                            bandWidth: bandWidth,
                          ),
                        ),
                      ),
                    ],
                  );
                },
              );
            },
          ),
        ),
      ),
    );
  }

  double lerpDouble(double min, double max, double t) => min + ((max - min) * t);
}

class _ProgressFill extends StatelessWidget {
  const _ProgressFill({
    required this.color,
    required this.shimmer,
    required this.highlightStartColor,
    required this.highlightEndColor,
    required this.bandLeft,
    required this.bandWidth,
  });

  final Color color;
  final bool shimmer;
  final Color highlightStartColor;
  final Color highlightEndColor;
  final double bandLeft;
  final double bandWidth;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(999),
        boxShadow: shimmer ? [BoxShadow(color: const Color(0x33FFFFFF), blurRadius: 8)] : null,
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(999),
        child: shimmer
            ? Stack(
                fit: StackFit.expand,
                children: [
                  Positioned(
                    left: bandLeft,
                    width: bandWidth,
                    top: 0,
                    bottom: 0,
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          colors: [highlightStartColor, const Color(0xFFFFFFFF), highlightEndColor],
                        ),
                      ),
                    ),
                  ),
                ],
              )
            : const SizedBox.expand(),
      ),
    );
  }
}
