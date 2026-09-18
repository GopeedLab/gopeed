import 'package:flutter/widgets.dart';

import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';

/// A labelled control whose caller chooses the layout for the available width.
class AppFormRow extends StatelessWidget {
  const AppFormRow({super.key, required this.label, required this.child, this.direction = Axis.horizontal});

  final String label;
  final Widget child;
  final Axis direction;

  @override
  Widget build(BuildContext context) {
    final title = Text(
      label,
      style: TextStyle(color: AppPalette.of(context).textPrimary, fontSize: 13, fontWeight: FontWeight.w500),
    );
    if (direction == Axis.vertical) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          title,
          const SizedBox(height: AppDesignTokens.space8),
          SizedBox(width: double.infinity, child: child),
        ],
      );
    }
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 104,
          child: Padding(padding: const EdgeInsets.only(top: 10), child: title),
        ),
        const SizedBox(width: AppDesignTokens.space12),
        Expanded(child: child),
      ],
    );
  }
}
