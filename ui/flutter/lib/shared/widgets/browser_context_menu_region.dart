import 'package:flutter/foundation.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';

import 'browser_context_menu_region_stub.dart'
    if (dart.library.js_interop) 'browser_context_menu_region_web.dart'
    as platform;

/// Keeps the browser menu out of an application's context-menu region on Web.
///
/// Captured themes carry the policy into shadcn's menu and submenu overlays.
/// Overlay regions defer to their children so a full-screen positioning layer
/// does not suppress the browser menu over the rest of the page.
class BrowserContextMenuRegion extends StatelessWidget {
  const BrowserContextMenuRegion({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (!kIsWeb) return child;
    return _ContextMenuTheme(
      child: _ContextMenuRegion(behavior: HitTestBehavior.translucent, child: child),
    );
  }
}

class _ContextMenuTheme extends InheritedTheme {
  const _ContextMenuTheme({required super.child});

  @override
  Widget wrap(BuildContext context, Widget child) {
    return _ContextMenuTheme(child: _ContextMenuRegion(child: child));
  }

  @override
  bool updateShouldNotify(_ContextMenuTheme oldWidget) => false;
}

class _ContextMenuRegion extends SingleChildRenderObjectWidget {
  const _ContextMenuRegion({this.behavior = HitTestBehavior.deferToChild, required super.child});

  final HitTestBehavior behavior;

  @override
  RenderObject createRenderObject(BuildContext context) {
    return _RenderContextMenuRegion(View.of(context).viewId, behavior);
  }

  @override
  void updateRenderObject(BuildContext context, _RenderContextMenuRegion renderObject) {
    renderObject
      ..viewId = View.of(context).viewId
      ..behavior = behavior;
  }
}

class _RenderContextMenuRegion extends RenderProxyBoxWithHitTestBehavior {
  _RenderContextMenuRegion(this._viewId, HitTestBehavior behavior) : super(behavior: behavior);

  int _viewId;

  set viewId(int value) {
    if (value == _viewId) return;
    if (attached) platform.unregisterRegion(this, _viewId);
    _viewId = value;
    if (attached) platform.registerRegion(this, _viewId);
  }

  @override
  void attach(PipelineOwner owner) {
    super.attach(owner);
    platform.registerRegion(this, _viewId);
  }

  @override
  void detach() {
    platform.unregisterRegion(this, _viewId);
    super.detach();
  }
}
