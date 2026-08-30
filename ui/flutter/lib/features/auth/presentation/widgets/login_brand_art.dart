import 'dart:math' as math;

import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/gopeed_app_mark.dart';

class LoginBrandArt extends StatelessWidget {
  const LoginBrandArt({super.key, this.compact = false});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: AlignmentDirectional.topStart,
          end: AlignmentDirectional.bottomEnd,
          colors: [palette.sideBg, Color.alphaBlend(palette.brandSoft, palette.sideBg), palette.cardBg],
        ),
      ),
      child: Stack(
        fit: StackFit.expand,
        children: [
          CustomPaint(
            painter: _TransferFlowPainter(palette: palette, compact: compact),
          ),
          PositionedDirectional(
            start: compact ? AppDesignTokens.space24 : AppDesignTokens.space32,
            top: compact ? AppDesignTokens.space16 : AppDesignTokens.space32,
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  'Gopeed',
                  style: TextStyle(
                    color: palette.textPrimary,
                    fontSize: compact ? 17 : 19,
                    fontWeight: FontWeight.w800,
                    letterSpacing: 0.4,
                  ),
                ),
              ],
            ),
          ),
          Center(child: _TransferOrbit(compact: compact)),
        ],
      ),
    );
  }
}

class _TransferOrbit extends StatelessWidget {
  const _TransferOrbit({required this.compact});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final size = compact ? 132.0 : 260.0;
    final coreSize = compact ? 62.0 : 96.0;
    return SizedBox.square(
      dimension: size,
      child: Stack(
        alignment: Alignment.center,
        clipBehavior: Clip.none,
        children: [
          CustomPaint(
            size: Size.square(size),
            painter: _OrbitPainter(palette: palette),
          ),
          Container(
            width: coreSize,
            height: coreSize,
            padding: EdgeInsets.all(compact ? 7 : 11),
            decoration: BoxDecoration(
              color: palette.cardBg,
              shape: BoxShape.circle,
              border: Border.all(color: palette.brand.withValues(alpha: 0.42)),
              boxShadow: [
                BoxShadow(color: palette.brand.withValues(alpha: 0.18), blurRadius: compact ? 18 : 32, spreadRadius: 2),
              ],
            ),
            child: GopeedAppMark(size: coreSize - (compact ? 14 : 22)),
          ),
          PositionedDirectional(
            top: compact ? 5 : 16,
            end: compact ? 3 : 20,
            child: _OrbitNode(icon: shad.LucideIcons.cloudDownload, compact: compact),
          ),
          PositionedDirectional(
            bottom: compact ? 7 : 20,
            start: compact ? 0 : 14,
            child: _OrbitNode(icon: shad.LucideIcons.hardDriveDownload, compact: compact),
          ),
          if (!compact)
            PositionedDirectional(
              bottom: 4,
              end: 38,
              child: _OrbitNode(icon: shad.LucideIcons.gauge, compact: false, small: true),
            ),
        ],
      ),
    );
  }
}

class _OrbitNode extends StatelessWidget {
  const _OrbitNode({required this.icon, required this.compact, this.small = false});

  final IconData icon;
  final bool compact;
  final bool small;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final dimension = small
        ? 34.0
        : compact
        ? 30.0
        : 42.0;
    return Container(
      width: dimension,
      height: dimension,
      decoration: BoxDecoration(
        color: palette.cardBg,
        borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius + 4),
        border: Border.all(color: palette.border),
        boxShadow: [
          BoxShadow(color: palette.textPrimary.withValues(alpha: 0.08), blurRadius: 12, offset: const Offset(0, 4)),
        ],
      ),
      child: Icon(
        icon,
        size: small
            ? 15
            : compact
            ? 14
            : 18,
        color: palette.brand,
      ),
    );
  }
}

class _TransferFlowPainter extends CustomPainter {
  const _TransferFlowPainter({required this.palette, required this.compact});

  final AppPalette palette;
  final bool compact;

  @override
  void paint(Canvas canvas, Size size) {
    final center = size.center(Offset.zero);
    final dotPaint = Paint()..color = palette.textMuted.withValues(alpha: 0.12);
    final gap = compact ? 24.0 : 28.0;
    for (var x = gap / 2; x < size.width; x += gap) {
      for (var y = gap / 2; y < size.height; y += gap) {
        canvas.drawCircle(Offset(x, y), 1, dotPaint);
      }
    }

    final paths = <Path>[
      Path()
        ..moveTo(-size.width * 0.08, size.height * 0.32)
        ..cubicTo(size.width * 0.20, size.height * 0.18, center.dx * 0.65, center.dy * 0.72, center.dx, center.dy),
      Path()
        ..moveTo(size.width * 1.08, size.height * 0.72)
        ..cubicTo(size.width * 0.78, size.height * 0.84, center.dx * 1.35, center.dy * 1.18, center.dx, center.dy),
      Path()
        ..moveTo(size.width * 0.18, size.height * 1.06)
        ..cubicTo(size.width * 0.26, size.height * 0.72, center.dx * 0.72, center.dy * 1.28, center.dx, center.dy),
    ];
    for (var index = 0; index < paths.length; index++) {
      canvas.drawPath(
        paths[index],
        Paint()
          ..color = (index == 0 ? palette.brand : palette.textMuted).withValues(alpha: index == 0 ? 0.28 : 0.13)
          ..style = PaintingStyle.stroke
          ..strokeWidth = index == 0 ? 1.8 : 1.0,
      );
    }

    final glowPaint = Paint()
      ..shader = RadialGradient(
        colors: [palette.brand.withValues(alpha: 0.13), palette.brand.withValues(alpha: 0)],
      ).createShader(Rect.fromCircle(center: center, radius: compact ? 90 : 180));
    canvas.drawCircle(center, compact ? 90 : 180, glowPaint);
  }

  @override
  bool shouldRepaint(_TransferFlowPainter oldDelegate) =>
      oldDelegate.palette != palette || oldDelegate.compact != compact;
}

class _OrbitPainter extends CustomPainter {
  const _OrbitPainter({required this.palette});

  final AppPalette palette;

  @override
  void paint(Canvas canvas, Size size) {
    final center = size.center(Offset.zero);
    final radius = size.shortestSide * 0.39;
    canvas.drawCircle(
      center,
      radius,
      Paint()
        ..color = palette.brand.withValues(alpha: 0.26)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.2,
    );
    canvas.drawArc(
      Rect.fromCircle(center: center, radius: radius * 0.72),
      -math.pi * 0.85,
      math.pi * 1.38,
      false,
      Paint()
        ..color = palette.brand.withValues(alpha: 0.52)
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2.2
        ..strokeCap = StrokeCap.round,
    );
    for (final angle in [-0.72, 0.18, 0.76]) {
      final point = center + Offset(math.cos(angle * math.pi), math.sin(angle * math.pi)) * radius;
      canvas.drawCircle(point, 3.2, Paint()..color = palette.brand);
    }
  }

  @override
  bool shouldRepaint(_OrbitPainter oldDelegate) => oldDelegate.palette != palette;
}
