import 'package:flutter/widgets.dart';

/// Switches adjacent tabs after a deliberate horizontal swipe over the content.
class AppSwipeTabs<T> extends StatefulWidget {
  const AppSwipeTabs({
    super.key,
    required this.values,
    required this.selectedValue,
    required this.onSelected,
    required this.child,
    this.enabled = true,
  });

  final List<T> values;
  final T selectedValue;
  final ValueChanged<T> onSelected;
  final Widget child;
  final bool enabled;

  @override
  State<AppSwipeTabs<T>> createState() => _AppSwipeTabsState<T>();
}

class _AppSwipeTabsState<T> extends State<AppSwipeTabs<T>> {
  double _distance = 0;

  void _finish(DragEndDetails details) {
    final distance = _distance;
    _distance = 0;
    final velocity = details.primaryVelocity ?? 0;
    if (distance.abs() < 48 && (distance.abs() < 12 || velocity.abs() < 400)) return;
    final direction = velocity.abs() >= 400 ? velocity : distance;
    final index = widget.values.indexOf(widget.selectedValue);
    if (index < 0) return;
    final forward = Directionality.of(context) == TextDirection.rtl ? direction > 0 : direction < 0;
    final nextIndex = index + (forward ? 1 : -1);
    if (nextIndex >= 0 && nextIndex < widget.values.length) widget.onSelected(widget.values[nextIndex]);
  }

  @override
  Widget build(BuildContext context) => GestureDetector(
    behavior: HitTestBehavior.opaque,
    onHorizontalDragStart: widget.enabled ? (_) => _distance = 0 : null,
    onHorizontalDragUpdate: widget.enabled ? (details) => _distance += details.primaryDelta ?? 0 : null,
    onHorizontalDragEnd: widget.enabled ? _finish : null,
    onHorizontalDragCancel: widget.enabled ? () => _distance = 0 : null,
    child: widget.child,
  );
}
