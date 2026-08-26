import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

class ActionIconButton extends StatelessWidget {
  const ActionIconButton({super.key, required this.icon, required this.color, this.onTap});

  final IconData icon;
  final Color color;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return shad.IconButton.ghost(
      onPressed: onTap,
      density: shad.ButtonDensity.compact,
      icon: Icon(icon, color: color, size: 18),
    );
  }
}
