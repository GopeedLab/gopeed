import 'dart:math' as math;

import 'package:flutter/widgets.dart';

import 'app_tooltip.dart';
import 'file_tree_horizontal_viewport.dart';

/// Keeps the full name scrollable, with an ellipsis at its visible trailing edge.
class FileTreeName extends StatelessWidget {
  const FileTreeName({
    super.key,
    required this.name,
    required this.style,
    required this.controller,
    required this.contentWidth,
    this.treeContentOffset,
  });

  final String name;
  final TextStyle style;
  final ScrollController controller;
  final double contentWidth;

  /// The name's leading position on the canvas when the whole tree scrolls.
  /// Null means that only the name scrolls, with the trailing columns frozen.
  final double? treeContentOffset;

  @override
  Widget build(BuildContext context) {
    final treeViewportWidth = FileTreeHorizontalViewport.widthOf(context);
    final direction = Directionality.of(context);
    return AppTooltip(
      message: name,
      child: LayoutBuilder(
        builder: (context, constraints) => AnimatedBuilder(
          animation: controller,
          builder: (context, child) {
            final offset = controller.hasClients ? controller.offset : 0.0;
            final available = treeContentOffset == null
                ? constraints.maxWidth + offset
                : (treeViewportWidth ?? constraints.maxWidth) + offset - treeContentOffset!;
            final width = math.max(
              0.0,
              math.min(treeContentOffset == null ? contentWidth : constraints.maxWidth, available),
            );
            final text = SizedBox(
              width: width,
              child: Text(name, style: style, maxLines: 1, softWrap: false, overflow: TextOverflow.ellipsis),
            );
            if (treeContentOffset != null) {
              return Align(alignment: AlignmentDirectional.centerStart, child: text);
            }
            return ClipRect(
              child: OverflowBox(
                alignment: AlignmentDirectional.centerStart,
                minWidth: 0,
                maxWidth: double.infinity,
                child: Transform.translate(
                  offset: Offset(direction == TextDirection.rtl ? offset : -offset, 0),
                  child: text,
                ),
              ),
            );
          },
        ),
      ),
    );
  }
}
