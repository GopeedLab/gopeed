import 'package:flutter/material.dart' show Tooltip;
import 'package:flutter/widgets.dart';

/// The single product tooltip primitive.
///
/// Flutter's tooltip is used intentionally because the shadcn tooltip does not
/// currently inherit Gopeed's mixed Shadcn/Material theme consistently.
class AppTooltip extends StatelessWidget {
  const AppTooltip({super.key, required this.message, required this.child});

  final String message;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Tooltip(message: message, child: child);
  }
}
