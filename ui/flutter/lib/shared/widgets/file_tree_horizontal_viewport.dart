import 'dart:math' as math;

import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../theme/app_design_tokens.dart';

/// Scrolls the header and virtualized tree together without freezing columns.
class FileTreeHorizontalViewport extends StatelessWidget {
  const FileTreeHorizontalViewport({
    super.key,
    required this.keyPrefix,
    required this.controller,
    required this.contentWidth,
    required this.child,
  });

  final String keyPrefix;
  final ScrollController controller;
  final double contentWidth;
  final Widget child;

  static double? widthOf(BuildContext context) {
    return context.dependOnInheritedWidgetOfExactType<_FileTreeHorizontalScope>()?.width;
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final width = math.max(constraints.maxWidth, contentWidth);
        final canScroll = width > constraints.maxWidth + 0.5;
        return _FileTreeHorizontalScope(
          width: constraints.maxWidth,
          child: Scrollbar(
            key: ValueKey('$keyPrefix-horizontal-scrollbar'),
            controller: controller,
            thumbVisibility: canScroll,
            trackVisibility: canScroll,
            interactive: true,
            thickness: 6,
            scrollbarOrientation: ScrollbarOrientation.bottom,
            notificationPredicate: (notification) => notification.metrics.axis == Axis.horizontal,
            child: Padding(
              padding: EdgeInsets.only(bottom: canScroll ? AppDesignTokens.space12 : 0),
              child: SingleChildScrollView(
                key: ValueKey('$keyPrefix-horizontal-scroll'),
                controller: controller,
                scrollDirection: Axis.horizontal,
                child: SizedBox(width: width, child: child),
              ),
            ),
          ),
        );
      },
    );
  }
}

class _FileTreeHorizontalScope extends InheritedWidget {
  const _FileTreeHorizontalScope({required this.width, required super.child});

  final double width;

  @override
  bool updateShouldNotify(_FileTreeHorizontalScope oldWidget) => width != oldWidget.width;
}
