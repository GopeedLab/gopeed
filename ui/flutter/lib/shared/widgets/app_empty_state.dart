import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';
import 'app_tooltip.dart';

/// Shared empty/error surface used by task and extension views.
class AppEmptyState extends StatelessWidget {
  const AppEmptyState({super.key, required this.message, this.action, this.illustrationKey});

  final String message;
  final Widget? action;
  final Key? illustrationKey;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Center(
      child: SingleChildScrollView(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 360),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                CustomPaint(
                  key: illustrationKey,
                  size: const Size(148, 108),
                  painter: _EmptyStateIllustrationPainter(palette: palette),
                ),
                const SizedBox(height: 18),
                Text(
                  message,
                  textAlign: TextAlign.center,
                  style: TextStyle(color: palette.textPrimary, fontSize: 17, fontWeight: FontWeight.w700),
                ),
                if (action != null) ...[const SizedBox(height: AppDesignTokens.space24), action!],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// A small outlined action icon beside a muted label, matching task guidance.
class AppEmptyStateAction extends StatelessWidget {
  const AppEmptyStateAction({
    super.key,
    required this.icon,
    required this.label,
    required this.onPressed,
    this.buttonKey,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  final Key? buttonKey;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        AppTooltip(
          message: label,
          child: SizedBox.square(
            dimension: 30,
            child: shad.IconButton.outline(
              key: buttonKey,
              density: shad.ButtonDensity.compact,
              onPressed: onPressed,
              icon: Icon(icon, size: 16),
            ),
          ),
        ),
        const SizedBox(width: 8),
        Flexible(
          child: Text(label, style: TextStyle(color: AppPalette.of(context).textMuted, fontSize: 13)),
        ),
      ],
    );
  }
}

class _EmptyStateIllustrationPainter extends CustomPainter {
  const _EmptyStateIllustrationPainter({required this.palette});

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
  bool shouldRepaint(_EmptyStateIllustrationPainter oldDelegate) {
    return oldDelegate.palette != palette;
  }
}
