import 'dart:js_interop';
import 'dart:ui' as ui;
import 'dart:ui_web' as ui_web;

import 'package:flutter/rendering.dart';
import 'package:web/web.dart' as web;

final _regions = <int, Set<RenderBox>>{};
JSFunction? _listener;

void registerRegion(RenderBox region, int viewId) {
  _regions.putIfAbsent(viewId, () => <RenderBox>{}).add(region);
  if (_listener != null) return;
  _listener = _onContextMenu.toJS;
  web.document.addEventListener('contextmenu', _listener, true.toJS);
}

void unregisterRegion(RenderBox region, int viewId) {
  final regions = _regions[viewId];
  regions?.remove(region);
  if (regions?.isEmpty ?? false) _regions.remove(viewId);
  if (_regions.isNotEmpty) return;
  web.document.removeEventListener('contextmenu', _listener, true.toJS);
  _listener = null;
}

void _onContextMenu(web.MouseEvent event) {
  if (!event.cancelable || event.defaultPrevented) return;
  for (final view in ui.PlatformDispatcher.instance.views) {
    final regions = _regions[view.viewId];
    if (regions == null) continue;
    final host = ui_web.views.getHostElement(view.viewId) as web.HTMLElement?;
    if (host == null || !event.composedPath().toDart.contains(host)) continue;
    final bounds = host.getBoundingClientRect();
    if (bounds.width <= 0 || bounds.height <= 0) continue;
    final x = event.clientX - bounds.left;
    final y = event.clientY - bounds.top;
    if (x < 0 || y < 0 || x >= bounds.width || y >= bounds.height) continue;

    // DOM client coordinates are CSS pixels. Map them through the public view
    // host bounds rather than assuming a DPR, page origin or browser zoom.
    final logicalSize = view.physicalSize / view.devicePixelRatio;
    final position = Offset(x * logicalSize.width / bounds.width, y * logicalSize.height / bounds.height);
    final result = HitTestResult();
    RendererBinding.instance.hitTestInView(result, position, view.viewId);
    if (result.path.any((entry) => regions.contains(entry.target))) {
      event.preventDefault();
      return;
    }
  }
}
