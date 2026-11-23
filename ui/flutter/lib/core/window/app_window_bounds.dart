import 'dart:math' as math;

import 'package:flutter/widgets.dart';
import 'package:screen_retriever/screen_retriever.dart';

Future<Rect> fitWindowHeight(
  Rect current, {
  required double preferredHeight,
  required double minimumHeight,
  double? windowScale,
}) async {
  final displays = await _getDisplays();
  Rect? workArea;
  var largestOverlap = -1.0;
  var nearestDistance = double.infinity;
  for (final display in displays) {
    final position = display.visiblePosition;
    if (position == null) continue;
    final size = display.visibleSize ?? display.size;
    // Windows reports each monitor in its own scale and the window in its current scale.
    final scale = windowScale == null ? 1.0 : (display.scaleFactor ?? windowScale) / windowScale;
    final area = (position * scale) & (size * scale);
    if (area.isEmpty) continue;
    final intersection = current.intersect(area);
    final overlap = intersection.isEmpty ? 0.0 : intersection.width * intersection.height;
    final center = current.center;
    final distance = (center - Offset(center.dx.clamp(area.left, area.right), center.dy.clamp(area.top, area.bottom)))
        .distanceSquared;
    if (overlap > largestOverlap || (overlap == largestOverlap && distance < nearestDistance)) {
      largestOverlap = overlap;
      nearestDistance = distance;
      workArea = area;
    }
  }
  if (workArea == null) return current.topLeft & Size(current.width, preferredHeight);

  final minHeight = math.min(minimumHeight, workArea.height);
  final top = current.top.clamp(workArea.top, workArea.bottom - minHeight);
  final height = math.max(1.0, math.min(preferredHeight, (workArea.bottom - top).floorToDouble()));
  return Rect.fromLTWH(current.left, top, current.width, height);
}

Future<List<Display>> _getDisplays() async {
  try {
    return await screenRetriever.getAllDisplays();
  } on Exception {
    return const [];
  }
}
