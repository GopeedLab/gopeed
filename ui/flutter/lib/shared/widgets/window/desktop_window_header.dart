import 'package:flutter/foundation.dart' show TargetPlatform, defaultTargetPlatform;
import 'package:flutter/material.dart' show Colors;
import 'package:flutter/widgets.dart';
import 'package:window_manager/window_manager.dart';

import '../../../core/window/app_window_chrome.dart';
import '../../theme/app_design_tokens.dart';
import '../../theme/app_palette.dart';

class DesktopWindowHeader extends StatelessWidget {
  const DesktopWindowHeader({super.key});

  @override
  Widget build(BuildContext context) {
    if (!AppWindowChrome.usesCustomChrome) {
      return const SizedBox.shrink();
    }

    return SizedBox(
      height: AppDesignTokens.windowHeaderHeight,
      child: const Row(
        children: [
          Expanded(child: DragToMoveArea(child: SizedBox.expand())),
          WindowCaptionControls(),
        ],
      ),
    );
  }
}

class WindowCaptionControls extends StatefulWidget {
  const WindowCaptionControls({super.key});

  @override
  State<WindowCaptionControls> createState() => _WindowCaptionControlsState();
}

class _WindowCaptionControlsState extends State<WindowCaptionControls> with WindowListener {
  static const double _captionGlyphSize = 20;
  bool _isMaximized = false;

  @override
  void initState() {
    super.initState();
    windowManager.addListener(this);
    _refreshState();
  }

  @override
  void dispose() {
    windowManager.removeListener(this);
    super.dispose();
  }

  @override
  void onWindowMaximize() => _refreshState();

  @override
  void onWindowUnmaximize() => _refreshState();

  Future<void> _refreshState() async {
    final value = await windowManager.isMaximized();
    if (mounted) {
      setState(() => _isMaximized = value);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!AppWindowChrome.usesCustomChrome) {
      return const SizedBox.shrink();
    }

    final palette = AppPalette.of(context);
    final textColor = palette.textSecondary;
    final isLinux = defaultTargetPlatform == TargetPlatform.linux;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      mainAxisSize: MainAxisSize.min,
      children: [
        _CaptionControlButton(
          icon: const _MinimizeGlyph(),
          foreground: textColor,
          size: isLinux ? 28 : 30,
          radius: isLinux ? 999 : 4,
          background: isLinux ? const Color(0x0DFFFFFF) : Colors.transparent,
          hoverBackground: isLinux ? const Color(0x1AE95420) : palette.captionButtonHover,
          hoverForeground: textColor,
          onPressed: windowManager.minimize,
        ),
        SizedBox(width: isLinux ? 8 : 4),
        _CaptionControlButton(
          icon: _isMaximized ? const _RestoreGlyph() : const _MaximizeGlyph(),
          foreground: textColor,
          size: isLinux ? 28 : 30,
          radius: isLinux ? 999 : 4,
          background: isLinux ? const Color(0x0DFFFFFF) : Colors.transparent,
          hoverBackground: isLinux ? const Color(0x1AE95420) : palette.captionButtonHover,
          hoverForeground: textColor,
          onPressed: _isMaximized ? windowManager.unmaximize : windowManager.maximize,
        ),
        SizedBox(width: isLinux ? 8 : 4),
        _CaptionControlButton(
          icon: const _CloseGlyph(),
          foreground: isLinux ? const Color(0xFFE95420) : textColor,
          background: isLinux ? const Color(0x33E95420) : Colors.transparent,
          size: isLinux ? 28 : 30,
          radius: isLinux ? 999 : 4,
          hoverBackground: isLinux ? const Color(0xFFE95420) : const Color(0xFFE81123),
          hoverForeground: const Color(0xFFFFFFFF),
          onPressed: windowManager.close,
        ),
      ],
    );
  }
}

class _CaptionControlButton extends StatefulWidget {
  const _CaptionControlButton({
    required this.icon,
    required this.foreground,
    required this.onPressed,
    required this.size,
    required this.radius,
    required this.background,
    this.hoverBackground,
    this.hoverForeground,
  });

  final Widget icon;
  final Color foreground;
  final VoidCallback onPressed;
  final double size;
  final double radius;
  final Color background;
  final Color? hoverBackground;
  final Color? hoverForeground;

  @override
  State<_CaptionControlButton> createState() => _CaptionControlButtonState();
}

class _CaptionControlButtonState extends State<_CaptionControlButton> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final background = _hovered ? (widget.hoverBackground ?? palette.captionButtonHover) : widget.background;
    final foreground = _hovered ? (widget.hoverForeground ?? widget.foreground) : widget.foreground;

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: widget.onPressed,
        child: Container(
          width: widget.size,
          height: widget.size,
          alignment: Alignment.center,
          decoration: BoxDecoration(color: background, borderRadius: BorderRadius.circular(widget.radius)),
          child: IconTheme(
            data: IconThemeData(color: foreground),
            child: Center(child: widget.icon),
          ),
        ),
      ),
    );
  }
}

class _MinimizeGlyph extends StatelessWidget {
  const _MinimizeGlyph();

  @override
  Widget build(BuildContext context) {
    final color = IconTheme.of(context).color ?? Colors.white;
    final size = _WindowCaptionControlsState._captionGlyphSize;
    return SizedBox(
      width: size,
      height: size,
      child: Center(
        child: Container(
          width: size * 0.72,
          height: size <= 12 ? 1.2 : 1.6,
          decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(999)),
        ),
      ),
    );
  }
}

class _MaximizeGlyph extends StatelessWidget {
  const _MaximizeGlyph();

  @override
  Widget build(BuildContext context) {
    final color = IconTheme.of(context).color ?? Colors.white;
    final size = _WindowCaptionControlsState._captionGlyphSize;
    final boxSize = size * 0.62;
    return SizedBox(
      width: size,
      height: size,
      child: Center(
        child: Container(
          width: boxSize,
          height: boxSize,
          decoration: BoxDecoration(
            border: Border.all(color: color, width: size <= 12 ? 0.9 : 0.95),
            borderRadius: BorderRadius.circular(1),
          ),
        ),
      ),
    );
  }
}

class _RestoreGlyph extends StatelessWidget {
  const _RestoreGlyph();

  @override
  Widget build(BuildContext context) {
    final color = IconTheme.of(context).color ?? Colors.white;
    final size = _WindowCaptionControlsState._captionGlyphSize;
    final boxSize = size * 0.56;
    final topBoxLeft = size * 0.40;
    final topBoxTop = size * 0.16;
    final bottomBoxLeft = size * 0.16;
    final bottomBoxTop = size * 0.40;
    final borderWidth = size <= 12 ? 1.0 : 1.15;
    return SizedBox(
      width: size,
      height: size,
      child: Stack(
        children: [
          Positioned(
            left: topBoxLeft,
            top: topBoxTop,
            child: Container(
              width: boxSize,
              height: boxSize,
              decoration: BoxDecoration(
                border: Border.all(color: color, width: borderWidth),
                borderRadius: BorderRadius.circular(1),
              ),
            ),
          ),
          Positioned(
            left: bottomBoxLeft,
            top: bottomBoxTop,
            child: Container(
              width: boxSize,
              height: boxSize,
              decoration: BoxDecoration(
                color: Colors.transparent,
                border: Border.all(color: color, width: borderWidth),
                borderRadius: BorderRadius.circular(1),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _CloseGlyph extends StatelessWidget {
  const _CloseGlyph();

  @override
  Widget build(BuildContext context) {
    final color = IconTheme.of(context).color ?? Colors.white;
    return SizedBox(
      width: _WindowCaptionControlsState._captionGlyphSize,
      height: _WindowCaptionControlsState._captionGlyphSize,
      child: CustomPaint(painter: _CrossPainter(color)),
    );
  }
}

class _CrossPainter extends CustomPainter {
  const _CrossPainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = size.width <= 12 ? 1.2 : 1.8
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(
      Offset(size.width * 0.28, size.height * 0.28),
      Offset(size.width * 0.72, size.height * 0.72),
      paint,
    );
    canvas.drawLine(
      Offset(size.width * 0.72, size.height * 0.28),
      Offset(size.width * 0.28, size.height * 0.72),
      paint,
    );
  }

  @override
  bool shouldRepaint(covariant _CrossPainter oldDelegate) {
    return oldDelegate.color != color;
  }
}
