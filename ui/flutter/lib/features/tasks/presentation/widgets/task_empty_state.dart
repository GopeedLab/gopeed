import 'package:flutter/widgets.dart';

import '../../../../shared/theme/app_palette.dart';
import '../../../../l10n/l10n.dart';

class TaskEmptyState extends StatelessWidget {
  const TaskEmptyState({super.key, this.message});

  final String? message;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 360),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              CustomPaint(
                key: const ValueKey('task-empty-illustration'),
                size: const Size(148, 108),
                painter: _TaskEmptyIllustrationPainter(palette: palette),
              ),
              const SizedBox(height: 18),
              Text(
                message ?? context.l10n.emptyTaskList,
                style: TextStyle(color: palette.textPrimary, fontSize: 17, fontWeight: FontWeight.w700),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _TaskEmptyIllustrationPainter extends CustomPainter {
  const _TaskEmptyIllustrationPainter({required this.palette});

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
  bool shouldRepaint(_TaskEmptyIllustrationPainter oldDelegate) {
    return oldDelegate.palette != palette;
  }
}
