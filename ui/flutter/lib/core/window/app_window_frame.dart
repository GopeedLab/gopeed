import 'package:flutter/foundation.dart' show TargetPlatform, defaultTargetPlatform;
import 'package:flutter/material.dart' show Colors;
import 'package:flutter/widgets.dart';
import 'package:window_manager/window_manager.dart';

import '../../shared/theme/app_palette.dart';
import '../../shared/widgets/window/desktop_window_header.dart';
import 'app_window_chrome.dart';

const double kWindowCornerRadius = 12;

class AppWindowFrame extends StatelessWidget {
  const AppWindowFrame({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    Widget framedChild = child;

    // System insets apply to mobile devices at every responsive breakpoint.
    // SafeArea also consumes the padding so page-level SafeAreas do not add it twice.
    if (defaultTargetPlatform == TargetPlatform.android || defaultTargetPlatform == TargetPlatform.iOS) {
      framedChild = ColoredBox(
        color: AppPalette.of(context).bg,
        child: SafeArea(child: framedChild),
      );
    }

    if (AppWindowChrome.usesCustomChrome) {
      framedChild = Stack(
        children: [
          Positioned.fill(child: child),
          const Positioned(top: 0, left: 0, right: 0, child: DesktopWindowHeader()),
        ],
      );
    }

    if (AppWindowChrome.clipsRoundedCorners) {
      framedChild = ClipRRect(borderRadius: BorderRadius.circular(kWindowCornerRadius), child: framedChild);
    }

    if (AppWindowChrome.clipsRoundedCorners) {
      framedChild = DecoratedBox(
        decoration: const BoxDecoration(color: Colors.transparent),
        child: framedChild,
      );
    }

    if (AppWindowChrome.usesCustomChrome) {
      framedChild = DragToResizeArea(resizeEdgeSize: 6, child: framedChild);
    }

    return framedChild;
  }
}
