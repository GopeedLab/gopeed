import 'package:flutter/material.dart' show Colors;
import 'package:flutter/widgets.dart';

import '../../shared/widgets/window/desktop_window_header.dart';
import 'app_window_chrome.dart';

const double kWindowCornerRadius = 12;

class AppWindowFrame extends StatelessWidget {
  const AppWindowFrame({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    Widget framedChild = child;

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
      return DecoratedBox(
        decoration: const BoxDecoration(color: Colors.transparent),
        child: framedChild,
      );
    }

    return framedChild;
  }
}
