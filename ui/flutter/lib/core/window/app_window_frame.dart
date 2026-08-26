import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' show Colors;
import 'package:flutter/widgets.dart';

import '../../shared/widgets/window/desktop_window_header.dart';

const double kWindowCornerRadius = 12;

class AppWindowFrame extends StatelessWidget {
  const AppWindowFrame({super.key, required this.child});

  final Widget child;

  bool get _supportsCustomChrome =>
      !kIsWeb && (defaultTargetPlatform == TargetPlatform.windows || defaultTargetPlatform == TargetPlatform.linux);

  bool get _clipRoundedCorners => !kIsWeb && defaultTargetPlatform == TargetPlatform.windows;

  @override
  Widget build(BuildContext context) {
    Widget framedChild = child;

    if (_supportsCustomChrome) {
      framedChild = Stack(
        children: [
          Positioned.fill(child: child),
          const Positioned(top: 0, left: 0, right: 0, child: DesktopWindowHeader()),
        ],
      );
    }

    if (_clipRoundedCorners) {
      framedChild = ClipRRect(borderRadius: BorderRadius.circular(kWindowCornerRadius), child: framedChild);
    }

    if (_clipRoundedCorners) {
      return DecoratedBox(
        decoration: const BoxDecoration(color: Colors.transparent),
        child: framedChild,
      );
    }

    return framedChild;
  }
}
