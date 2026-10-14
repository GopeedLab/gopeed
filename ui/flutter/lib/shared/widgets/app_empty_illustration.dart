import 'package:flutter/widgets.dart';

import '../theme/app_palette.dart';

enum AppEmptyIllustrationType { inbox, puzzle, search, connectionError, documentError }

/// Context-specific drawings share the same muted colors and rounded strokes.
class AppEmptyIllustration extends StatelessWidget {
  const AppEmptyIllustration({super.key, required this.type});

  final AppEmptyIllustrationType type;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return CustomPaint(
      size: const Size(148, 108),
      painter: type == AppEmptyIllustrationType.inbox
          ? _InboxIllustrationPainter(palette: palette)
          : _ContextIllustrationPainter(palette: palette, type: type),
    );
  }
}

class _ContextIllustrationPainter extends CustomPainter {
  const _ContextIllustrationPainter({required this.palette, required this.type});

  final AppPalette palette;
  final AppEmptyIllustrationType type;

  Paint get _outline => Paint()
    ..color = palette.textSecondary.withValues(alpha: 0.72)
    ..style = PaintingStyle.stroke
    ..strokeWidth = 1.8
    ..strokeCap = StrokeCap.round
    ..strokeJoin = StrokeJoin.round;

  Paint get _detail => Paint()
    ..color = palette.textMuted.withValues(alpha: 0.34)
    ..style = PaintingStyle.stroke
    ..strokeWidth = 2
    ..strokeCap = StrokeCap.round;

  Paint get _fill => Paint()..color = palette.surfaceSoft;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.save();
    canvas.scale(size.width / 148, size.height / 108);
    switch (type) {
      case AppEmptyIllustrationType.puzzle:
        _paintPuzzle(canvas);
      case AppEmptyIllustrationType.search:
        _paintSearch(canvas);
      case AppEmptyIllustrationType.connectionError:
        _paintConnectionError(canvas);
      case AppEmptyIllustrationType.documentError:
        _paintDocumentError(canvas);
      case AppEmptyIllustrationType.inbox:
        break;
    }
    canvas.restore();
  }

  void _paintPuzzle(Canvas canvas) {
    canvas.translate(9, 0);
    final piece = Path()
      ..moveTo(42, 29)
      ..lineTo(62, 29)
      ..cubicTo(55, 10, 86, 10, 79, 29)
      ..lineTo(98, 29)
      ..quadraticBezierTo(104, 29, 104, 35)
      ..lineTo(104, 49)
      ..cubicTo(84, 42, 84, 73, 104, 66)
      ..lineTo(104, 86)
      ..quadraticBezierTo(104, 92, 98, 92)
      ..lineTo(79, 92)
      ..cubicTo(86, 73, 55, 73, 62, 92)
      ..lineTo(42, 92)
      ..quadraticBezierTo(36, 92, 36, 86)
      ..lineTo(36, 66)
      ..cubicTo(17, 73, 17, 42, 36, 49)
      ..lineTo(36, 35)
      ..quadraticBezierTo(36, 29, 42, 29)
      ..close();
    canvas.drawPath(piece, _fill);
    canvas.drawPath(piece, _outline);
  }

  void _paintSearch(Canvas canvas) {
    final lens = const Offset(64, 45);
    canvas.drawCircle(lens, 29, _fill);
    canvas.drawCircle(lens, 29, _outline);
    canvas.drawLine(const Offset(85, 67), const Offset(115, 96), _outline..strokeWidth = 4);
    canvas.drawLine(const Offset(53, 45), const Offset(75, 45), _detail);
    canvas.drawLine(const Offset(27, 85), const Offset(41, 85), _detail);
    canvas.drawCircle(const Offset(112, 23), 3, Paint()..color = palette.border);
  }

  void _paintConnectionError(Canvas canvas) {
    final cloud = Path()
      ..moveTo(40, 70)
      ..cubicTo(21, 70, 19, 43, 39, 41)
      ..cubicTo(40, 14, 77, 8, 91, 32)
      ..cubicTo(111, 23, 129, 40, 121, 56)
      ..cubicTo(133, 66, 119, 79, 108, 74)
      ..lineTo(40, 74)
      ..close();
    canvas.drawPath(cloud, _fill);
    canvas.drawPath(cloud, _outline);
    canvas.drawLine(const Offset(74, 74), const Offset(74, 82), _detail);
    canvas.drawLine(const Offset(48, 96), const Offset(63, 96), _outline);
    canvas.drawLine(const Offset(86, 96), const Offset(101, 96), _outline);
    canvas.drawLine(const Offset(71, 91), const Offset(77, 101), _detail);
    canvas.drawCircle(const Offset(45, 96), 2, _fill);
    canvas.drawCircle(const Offset(104, 96), 2, _fill);
  }

  void _paintDocumentError(Canvas canvas) {
    final document = Path()
      ..moveTo(47, 12)
      ..lineTo(89, 12)
      ..lineTo(107, 30)
      ..lineTo(107, 89)
      ..quadraticBezierTo(107, 96, 100, 96)
      ..lineTo(47, 96)
      ..quadraticBezierTo(40, 96, 40, 89)
      ..lineTo(40, 19)
      ..quadraticBezierTo(40, 12, 47, 12)
      ..close();
    canvas.drawPath(document, _fill);
    canvas.drawPath(document, _outline);
    canvas.drawPath(
      Path()
        ..moveTo(89, 12)
        ..lineTo(89, 30)
        ..lineTo(107, 30),
      _detail,
    );
    for (final y in [43.0, 56.0, 69.0]) {
      canvas.drawLine(Offset(53, y), Offset(y == 69 ? 74 : 91, y), _detail);
    }
    canvas.drawCircle(const Offset(107, 85), 16, Paint()..color = palette.cardBg);
    canvas.drawCircle(const Offset(107, 85), 16, _outline);
    canvas.drawLine(const Offset(107, 77), const Offset(107, 86), _outline);
    canvas.drawCircle(const Offset(107, 92), 1.2, Paint()..color = palette.textSecondary.withValues(alpha: 0.72));
  }

  @override
  bool shouldRepaint(_ContextIllustrationPainter oldDelegate) =>
      oldDelegate.palette != palette || oldDelegate.type != type;
}

class _InboxIllustrationPainter extends CustomPainter {
  const _InboxIllustrationPainter({required this.palette});

  final AppPalette palette;

  @override
  void paint(Canvas canvas, Size size) {
    final borderPaint = Paint()
      ..color = palette.border
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.4;
    final sheetRect = RRect.fromRectAndRadius(Rect.fromLTWH(size.width / 2 - 34, 6, 68, 72), const Radius.circular(7));

    canvas.drawRRect(sheetRect.shift(const Offset(0, 3)), Paint()..color = palette.textMuted.withValues(alpha: 0.06));
    canvas.drawRRect(sheetRect, Paint()..color = palette.cardBg);
    canvas.drawRRect(sheetRect, borderPaint);

    final rowPaint = Paint()
      ..color = palette.textMuted.withValues(alpha: 0.34)
      ..strokeWidth = 2
      ..strokeCap = StrokeCap.round;
    final centerX = size.width / 2;
    for (var row = 0; row < 2; row++) {
      final y = 20.0 + row * 14;
      canvas.drawCircle(Offset(centerX - 22, y), 2.5, Paint()..color = palette.surfaceSoft);
      canvas.drawLine(Offset(centerX - 14, y), Offset(centerX + 21, y), rowPaint);
    }

    final emptyBadgePaint = Paint()
      ..color = palette.surfaceSoft
      ..style = PaintingStyle.fill;
    canvas.drawCircle(Offset(centerX, 56), 11, emptyBadgePaint);
    canvas.drawCircle(Offset(centerX, 56), 11, borderPaint);
    canvas.drawLine(Offset(centerX - 4, 56), Offset(centerX + 4, 56), rowPaint);

    final trayPath = Path()
      ..moveTo(26, 63)
      ..lineTo(49, 63)
      ..lineTo(57, 75)
      ..quadraticBezierTo(59, 78, 63, 78)
      ..lineTo(85, 78)
      ..quadraticBezierTo(89, 78, 91, 75)
      ..lineTo(99, 63)
      ..lineTo(122, 63)
      ..lineTo(115, 96)
      ..quadraticBezierTo(114, 101, 108, 101)
      ..lineTo(40, 101)
      ..quadraticBezierTo(34, 101, 33, 96)
      ..close();
    canvas.drawPath(trayPath, Paint()..color = palette.surfaceSoft);
    final trayPaint = Paint()
      ..color = palette.textSecondary.withValues(alpha: 0.72)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.8
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;
    canvas.drawPath(trayPath, trayPaint);
  }

  @override
  bool shouldRepaint(_InboxIllustrationPainter oldDelegate) {
    return oldDelegate.palette != palette;
  }
}
