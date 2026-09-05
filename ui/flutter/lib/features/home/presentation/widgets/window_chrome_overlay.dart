import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:window_manager/window_manager.dart';

import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/window/desktop_window_header.dart';
import 'layout_constants.dart';

class WindowChromeOverlay extends StatelessWidget {
  const WindowChromeOverlay({super.key});

  @override
  Widget build(BuildContext context) {
    final isWindowsDesktop = !kIsWeb && defaultTargetPlatform == TargetPlatform.windows;

    return SizedBox(
      height: kWindowTaskbarHeight,
      child: Row(
        children: [
          SizedBox(
            width: 64 + 264,
            child: isWindowsDesktop ? const DragToMoveArea(child: SizedBox.expand()) : const SizedBox.shrink(),
          ),
          const Expanded(child: _WindowTaskbar()),
        ],
      ),
    );
  }
}

class _WindowTaskbar extends StatelessWidget {
  const _WindowTaskbar();

  @override
  Widget build(BuildContext context) {
    final isWindowsDesktop = !kIsWeb && defaultTargetPlatform == TargetPlatform.windows;
    final palette = AppPalette.of(context);

    return Container(
      color: palette.bg,
      child: Row(
        children: [
          Expanded(child: isWindowsDesktop ? const DragToMoveArea(child: SizedBox.expand()) : const SizedBox.shrink()),
          if (isWindowsDesktop) const WindowCaptionControls(),
        ],
      ),
    );
  }
}
