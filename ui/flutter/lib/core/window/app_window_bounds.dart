import 'dart:math' as math;

import 'package:flutter/widgets.dart';
import 'package:screen_retriever/screen_retriever.dart';

Future<double> expandedWindowHeight(Rect current, double height) async {
  if (height <= current.height) return current.height;

  Rect? workArea;
  var largestOverlap = 0.0;
  for (final display in await screenRetriever.getAllDisplays()) {
    final position = display.visiblePosition;
    if (position == null) continue;
    final area = position & (display.visibleSize ?? display.size);
    final intersection = current.intersect(area);
    if (intersection.isEmpty) continue;
    final overlap = intersection.width * intersection.height;
    if (overlap > largestOverlap) {
      largestOverlap = overlap;
      workArea = area;
    }
  }

  if (workArea == null) return height;
  return math.max(current.height, math.min(height, workArea.bottom - current.top));
}
