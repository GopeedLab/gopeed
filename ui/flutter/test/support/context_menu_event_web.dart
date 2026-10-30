import 'dart:ui';
import 'dart:ui_web' as ui_web;

import 'package:web/web.dart' as web;

/// Exercises DOM cancellation, independently of Flutter's global menu flag.
/// Synthetic events cannot prove whether a browser's native menu is visible.
bool dispatchContextMenu(Offset position, int viewId) {
  final view = PlatformDispatcher.instance.views.singleWhere((view) => view.viewId == viewId);
  final host = ui_web.views.getHostElement(viewId) as web.HTMLElement;
  final bounds = host.getBoundingClientRect();
  final logicalSize = view.physicalSize / view.devicePixelRatio;
  final event = web.MouseEvent(
    'contextmenu',
    web.MouseEventInit(
      bubbles: true,
      cancelable: true,
      button: 2,
      clientX: (bounds.left + position.dx * bounds.width / logicalSize.width).round(),
      clientY: (bounds.top + position.dy * bounds.height / logicalSize.height).round(),
    ),
  );
  host.dispatchEvent(event);
  return event.defaultPrevented;
}
