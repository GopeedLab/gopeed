import 'package:flutter/widgets.dart';

import '../theme/app_palette.dart';

class GopeedAppMark extends StatelessWidget {
  const GopeedAppMark({super.key, this.size = 24});

  final double size;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Semantics(
      label: 'Gopeed',
      image: true,
      child: SizedBox.square(
        dimension: size,
        child: CustomPaint(
          painter: _GopeedAppMarkPainter(
            background: palette.primaryActionBg,
            foreground: palette.primaryActionForeground,
          ),
        ),
      ),
    );
  }
}

class _GopeedAppMarkPainter extends CustomPainter {
  const _GopeedAppMarkPainter({required this.background, required this.foreground});

  final Color background;
  final Color foreground;

  @override
  void paint(Canvas canvas, Size size) {
    final side = size.shortestSide;
    final bounds = Rect.fromCenter(center: size.center(Offset.zero), width: side, height: side).deflate(side * 0.02);
    final circle = Path()..addOval(bounds);
    final backgroundPaint = Paint()
      ..color = background
      ..isAntiAlias = true;
    final foregroundPaint = Paint()
      ..color = foreground
      ..isAntiAlias = true;
    canvas.drawPath(circle, backgroundPaint);
    canvas.save();
    canvas.clipPath(circle, doAntiAlias: true);

    Offset point(double x, double y) {
      return Offset(bounds.left + bounds.width * x, bounds.top + bounds.height * y);
    }

    final cloud = Path()
      ..moveTo(point(-0.02, 0.83).dx, point(-0.02, 0.83).dy)
      ..cubicTo(
        point(0.04, 0.78).dx,
        point(0.04, 0.78).dy,
        point(0.09, 0.75).dx,
        point(0.09, 0.75).dy,
        point(0.15, 0.73).dx,
        point(0.15, 0.73).dy,
      )
      ..cubicTo(
        point(0.12, 0.66).dx,
        point(0.12, 0.66).dy,
        point(0.13, 0.60).dx,
        point(0.13, 0.60).dy,
        point(0.20, 0.57).dx,
        point(0.20, 0.57).dy,
      )
      ..cubicTo(
        point(0.27, 0.54).dx,
        point(0.27, 0.54).dy,
        point(0.31, 0.58).dx,
        point(0.31, 0.58).dy,
        point(0.35, 0.62).dx,
        point(0.35, 0.62).dy,
      )
      ..cubicTo(
        point(0.46, 0.42).dx,
        point(0.46, 0.42).dy,
        point(0.66, 0.40).dx,
        point(0.66, 0.40).dy,
        point(0.78, 0.50).dx,
        point(0.78, 0.50).dy,
      )
      ..cubicTo(
        point(0.86, 0.57).dx,
        point(0.86, 0.57).dy,
        point(0.87, 0.68).dx,
        point(0.87, 0.68).dy,
        point(0.86, 0.75).dx,
        point(0.86, 0.75).dy,
      )
      ..cubicTo(
        point(0.92, 0.76).dx,
        point(0.92, 0.76).dy,
        point(0.98, 0.79).dx,
        point(0.98, 0.79).dy,
        point(1.03, 0.84).dx,
        point(1.03, 0.84).dy,
      );
    canvas.drawPath(
      cloud,
      Paint()
        ..color = foreground
        ..style = PaintingStyle.stroke
        ..strokeWidth = side * 0.05
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round
        ..isAntiAlias = true,
    );

    final arrow = Path()
      ..addRect(Rect.fromPoints(point(0.41, 0.66), point(0.59, 0.86)))
      ..moveTo(point(0.31, 0.85).dx, point(0.31, 0.85).dy)
      ..lineTo(point(0.69, 0.85).dx, point(0.69, 0.85).dy)
      ..lineTo(point(0.50, 1.02).dx, point(0.50, 1.02).dy)
      ..close();
    canvas.drawPath(arrow, foregroundPaint);
    canvas.restore();
  }

  @override
  bool shouldRepaint(_GopeedAppMarkPainter oldDelegate) {
    return background != oldDelegate.background || foreground != oldDelegate.foreground;
  }
}
